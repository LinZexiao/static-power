package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	sapi "static-power/api"
	"static-power/server"
	"sync"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/urfave/cli/v2"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
)

func main() {
	app := &cli.App{
		Name:                 "static-power",
		Suggest:              true,
		EnableBashCompletion: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name: "listen",
				Aliases: []string{
					"l",
				},
				Value: "127.0.0.1:8090",
				Usage: "listen address",
			},
		},
		Commands: []*cli.Command{
			{
				Name: "daemon",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "node",
						Usage: "entry point for a filecoin node",
					},
					&cli.StringFlag{
						Name:  "dsn",
						Usage: "database connection string",
					},
				},
			},
		},
	}
	app.Setup()
	if err := app.Run(os.Args); err != nil {
		fmt.Println(err)
		return
	}
}

var daemonCmd = &cli.Command{
	Name: "daemon",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "dsn",
			Usage: "database connection string",
		},
	},
	Action: func(c *cli.Context) error {
		var db *gorm.DB
		var err error

		dsn := c.String("dsn")
		listen := c.String("listen")

		if dsn == "" {
			db, err = gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
		} else {
			db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
		}
		if err != nil {
			log.Fatal(err)
		}

		a := sapi.NewApi(db)
		server.RegisterApi(a)
		server.Run(listen)
		return nil
	},
}

var updatePowerCmd = &cli.Command{
	Name: "update-peer",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "node-url",
			Usage: "entry point for a filecoin node",
		},
		&cli.StringFlag{
			Name:  "node-token",
			Usage: "token for a filecoin node",
		},
		&cli.BoolFlag{
			Name:  "update-peer",
			Usage: "update miner peer by the way",
			Value: true,
		},
	},
	Action: func(c *cli.Context) error {
		listen := c.String("listen")
		if listen != "" {
			server.SetHost(listen)
		}

		// get miner power peer and update
		url := c.String("node-url")
		token := c.String("node-token")

		if url == "" {
			log.Fatal("node-url is required")
		}
		if token == "" {
			log.Fatal("node-token is required")
		}

		node, closer, err := NewRpcClient(url, &token)
		if err != nil {
			return err
		}
		defer closer()

		miners, err := getMinerInfosWithMinPower(node)
		if err != nil {
			return err
		}

		for _, miner := range miners {
			if miner.Power != nil {
				server.UpdatePowerInfo(miner.Power)
			}
			if miner.Peer != nil {
				server.UpdatePeerInfo(miner.Peer)
			}
		}
		log.Println("update power info success")
		return nil
	},
}

var updateAgentCmd = &cli.Command{
	Name: "update-agent",
	Action: func(c *cli.Context) error {
		listen := c.String("listen")
		if listen != "" {
			server.SetHost(listen)
		}

		miners, err := server.GetMiners()
		if err != nil {
			return fmt.Errorf("get miners : %w", err)
		}

		agents := getAgentInfo(miners)

		for _, agent := range agents {
			server.UpdateAgentInfo(agent)
		}
		// get miner get agent
		return nil
	},
}

func getAgentInfo(miners []sapi.Miner) []*sapi.AgentInfo {
	ret := make([]*sapi.AgentInfo, 0, len(miners))
	var wg sync.WaitGroup

	wg.Add(len(miners))
	throttle := make(chan struct{}, 5000)
	for i := range miners {
		miner := &miners[i]
		throttle <- struct{}{}

		go func(miner *MinerInfo) {
			defer func() {
				wg.Done()
				<-throttle
				// manager.TrimOpenConns(ctx)
			}()

			ctx := context.Background()

			info := miner.Peer

			if info == nil || info.PeerId == "" || len(*info.Multiaddrs) == 0 {
				return
			}

			err := func() error {

				host, err := libp2p.New(libp2p.NoListenAddrs)
				if err != nil {
					return err
				}
				defer host.Close()

				addrInfo := peer.AddrInfo{
					ID:    peer.ID(info.PeerId),
					Addrs: []multiaddr.Multiaddr{},
				}

				for _, addr := range *info.Multiaddrs {
					maddr, err := multiaddr.NewMultiaddr(addr)
					if err != nil {
						return fmt.Errorf("parsing multiaddr %s: %w", addr, err)
					}
					addrInfo.Addrs = append(addrInfo.Addrs, maddr)
				}

				if err := host.Connect(ctx, addrInfo); err != nil {
					return fmt.Errorf("connecting to peer %s: %w", addrInfo.ID, err)
				}

				userAgentI, err := host.Peerstore().Get(addrInfo.ID, "AgentVersion")
				if err != nil {
					return fmt.Errorf("getting user agent for peer %s: %w", addrInfo.ID, err)
				}

				userAgent, ok := userAgentI.(string)
				if !ok {
					return fmt.Errorf("user agent for peer %s was not a string", addrInfo.ID)
				}

				agentInfo := &sapi.AgentInfo{
					MinerID: miner.ID,
					Name:    userAgent,
				}

				if agentInfo.Name != "" {
					return nil
				}

				if miner.Agent != nil && miner.Agent.Name != agentInfo.Name {
					ret = append(ret, agentInfo)
				}
				return nil
			}()

			if err != nil {
				log.Printf("error getting agent for miner %s: %s", miner.ID.String(), err)
				return
			}
		}(miner)
	}
	wg.Wait()
	return nil
}

type MinerInfo = sapi.Miner

func getMinerInfosWithMinPower(node api.FullNode) ([]*MinerInfo, error) {
	ret := make([]*MinerInfo, 0)
	ctx := context.Background()
	miners, err := node.StateListMiners(ctx, types.EmptyTSK)
	if err != nil {
		return nil, err
	}
	log.Println("Total SPs on chain: ", len(miners))

	var wg sync.WaitGroup
	wg.Add(len(miners))
	var lk sync.Mutex

	if len(miners) != 0 {

	}

	throttle := make(chan struct{}, 100)
	for i, miner := range miners {
		throttle <- struct{}{}
		go func(miner address.Address) {
			defer wg.Done()
			defer func() {
				<-throttle
			}()

			power, err := node.StateMinerPower(ctx, miner, types.EmptyTSK)
			if err != nil {
				panic(err)
			}

			if !power.HasMinPower {
				return
			}

			info, err := node.StateMinerInfo(ctx, miner, types.EmptyTSK)
			if err != nil {
				panic(err)
			}

			id, err := address.IDFromAddress(miner)
			if err != nil {
				log.Println("miner id error: ", err)
			}
			aid := abi.ActorID(id)

			rbp := sapi.Power(power.MinerPower.RawBytePower)
			qap := sapi.Power(power.MinerPower.QualityAdjPower)
			powerInfo := sapi.PowerInfo{
				MinerID:                  aid,
				RawBytePower:             &rbp,
				QualityAdjustedBytePower: &qap,
			}

			multiAddress := sapi.Multiaddrs{}
			for _, addr := range info.Multiaddrs {
				maddr, err := multiaddr.NewMultiaddrBytes(addr)
				if err != nil {
					log.Println("parse multiaddr error: ", err)
				}
				multiAddress = append(multiAddress, maddr.String())
			}
			peerInfo := sapi.PeerInfo{
				Multiaddrs: &multiAddress,
			}
			if info.PeerId != nil {
				peerInfo.PeerId = info.PeerId.String()
			}
			mi := &MinerInfo{
				ID:    aid,
				Power: &powerInfo,
				Peer:  &peerInfo,
			}

			lk.Lock()
			ret = append(ret, mi)
			lk.Unlock()

			if i == 0 {
				// add network power
				rbp := sapi.Power(power.TotalPower.RawBytePower)
				qap := sapi.Power(power.TotalPower.QualityAdjPower)
				powerInfo := sapi.PowerInfo{
					MinerID:                  sapi.NetWork,
					RawBytePower:             &rbp,
					QualityAdjustedBytePower: &qap,
				}
				mi := &MinerInfo{
					ID:    sapi.NetWork,
					Power: &powerInfo,
				}

				lk.Lock()
				ret = append(ret, mi)
				lk.Unlock()
			}
		}(miner)
	}

	wg.Wait()
	return ret, nil
}

func NewRpcClient(endpoint string, token *string) (api.FullNode, jsonrpc.ClientCloser, error) {
	requestHeader := http.Header{}
	if token != nil {
		requestHeader.Add("Authorization", "Bearer "+*token)
	}
	var res api.FullNodeStruct
	closer, err := jsonrpc.NewMergeClient(context.Background(), endpoint, "Filecoin", api.GetInternalStructs(&res), requestHeader)
	return &res, closer, err
}
