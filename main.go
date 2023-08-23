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
			daemonCmd,
			updatePowerCmd,
			updateAgentCmd,
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
			Name:  "node",
			Usage: "entry point for a filecoin node",
		},
		&cli.StringFlag{
			Name:  "token",
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
		url := c.String("node")
		token := c.String("token")

		if url == "" {
			log.Fatal("node url is required")
		}
		if token == "" {
			log.Fatal("node token is required")
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
				err := server.UpdatePowerInfo(miner.Power)
				if err != nil {
					log.Printf("update power info for(%d) : %s", miner.ID, err)
				}
				log.Printf("update power info for(%d) success , RBP(%s), QAP(%s) ", miner.ID, miner.Power.RawBytePower.String(), miner.Power.QualityAdjPower.String())
			}
			if miner.Peer != nil {
				err := server.UpdatePeerInfo(miner.Peer)
				if err != nil {
					log.Printf("update peer info for(%d) : %s", miner.ID, err)
				}
				log.Printf("update peer info for(%d) success , PeerId(%s), Multiaddrs.len(%d) ", miner.ID, miner.Peer.PeerId, len(*miner.Peer.Multiaddrs))
			}
		}
		log.Println("update power info success")
		return nil
	},
}

var updateAgentCmd = &cli.Command{
	Name: "update-agent",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "tag",
			Usage:    fmt.Sprintf("tag for agent, values available: [ %s, %s, %s ]", sapi.TagLocationHongKong, sapi.TagLocationJapan, sapi.TagLocationSingapore),
			Required: true,
		},
	},
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

		log.Printf("update (%d) agent info of (%d), ", len(agents), len(miners))
		for i := range agents {
			agent := agents[i]
			agent.Tag = c.String("tag")
			err := server.UpdateAgentInfo(agent)
			if err != nil {
				log.Printf("update agent info for(%d) : %s", agent.MinerID, err)
			}
			log.Printf("update agent info for(%d) success , Name(%s), Tag(%s)", agent.MinerID, agent.Name, agent.Tag)
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

			err := func() error {
				host, err := libp2p.New(libp2p.NoListenAddrs)
				if err != nil {
					return err
				}
				defer host.Close()

				if info == nil || info.PeerId == "" || len(*info.Multiaddrs) == 0 {
					return fmt.Errorf("no peer info")
				}
				if info.PeerId == "" {
					return fmt.Errorf("empty peer id")
				}
				if info.Multiaddrs == nil || len(*info.Multiaddrs) == 0 {
					return fmt.Errorf("empty multiaddrs")
				}

				peerId, err := peer.Decode(info.PeerId)
				if err != nil {
					return fmt.Errorf("decode peer id %s: %w", info.PeerId, err)
				}

				addrInfo := peer.AddrInfo{
					ID:    peerId,
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

				if agentInfo.Name == "" {
					return fmt.Errorf("user agent empty")
				}

				if miner.Agent != nil && miner.Agent.Name == agentInfo.Name {
					// will not
					// return fmt.Errorf("user agent (%s) not change", miner.Agent.Name)
					log.Printf("user agent (%s) not change", miner.Agent.Name)
				}

				ret = append(ret, agentInfo)
				return nil
			}()

			if err != nil {
				log.Printf("get agent for miner %s: %s", miner.ID.String(), err)
				return
			}

		}(miner)
	}
	wg.Wait()
	return ret
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

	// get network power
	if len(miners) != 0 {
		power, err := node.StateMinerPower(ctx, miners[0], types.EmptyTSK)
		if err != nil {
			panic(err)
		}
		rbp := sapi.Power(power.TotalPower.RawBytePower)
		qap := sapi.Power(power.TotalPower.QualityAdjPower)
		powerInfo := sapi.PowerInfo{
			MinerID:         sapi.NetWork,
			RawBytePower:    &rbp,
			QualityAdjPower: &qap,
		}
		mi := &MinerInfo{
			ID:    sapi.NetWork,
			Power: &powerInfo,
		}
		ret = append(ret, mi)
	}

	throttle := make(chan struct{}, 100)
	for i := range miners {
		miner := miners[i]
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
				MinerID:         aid,
				RawBytePower:    &rbp,
				QualityAdjPower: &qap,
			}

			mi := &MinerInfo{
				ID:    aid,
				Power: &powerInfo,
			}

			if info.PeerId != nil || len(info.Multiaddrs) > 0 {
				multiAddress := sapi.Multiaddrs{}
				for _, addr := range info.Multiaddrs {
					maddr, err := multiaddr.NewMultiaddrBytes(addr)
					if err != nil {
						log.Println("parse multiaddr error: ", err)
					}
					multiAddress = append(multiAddress, maddr.String())
				}
				peerInfo := sapi.PeerInfo{
					MinerID:    aid,
					PeerId:     info.PeerId.String(),
					Multiaddrs: &multiAddress,
				}
				mi.Peer = &peerInfo
			}

			lk.Lock()
			ret = append(ret, mi)
			lk.Unlock()
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
