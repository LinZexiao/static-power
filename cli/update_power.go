package cli

import (
	"log"
	"static-power/server"
	"sync"

	"github.com/multiformats/go-multiaddr"
	"github.com/urfave/cli/v2"

	"context"

	"net/http"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"

	sapi "static-power/api"
)

type MinerInfo = sapi.Miner

var UpdatePowerCmd = &cli.Command{
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

func NewRpcClient(endpoint string, token *string) (api.FullNode, jsonrpc.ClientCloser, error) {
	requestHeader := http.Header{}
	if token != nil {
		requestHeader.Add("Authorization", "Bearer "+*token)
	}
	var res api.FullNodeStruct
	closer, err := jsonrpc.NewMergeClient(context.Background(), endpoint, "Filecoin", api.GetInternalStructs(&res), requestHeader)
	return &res, closer, err
}

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
