package cli

import (
	"context"
	"fmt"
	"log"
	sapi "static-power/api"
	"static-power/server"
	"sync"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/urfave/cli/v2"
)

var UpdateAgentCmd = &cli.Command{
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
