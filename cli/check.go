package cli

import (
	"fmt"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/urfave/cli/v2"
)

var CheckCmd = &cli.Command{
	Name: "check",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "node",
			Usage:    "url for node service",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "token",
			Usage:    "token for node service",
			Required: true,
		},
	},
	Usage:     "get agent of miner",
	ArgsUsage: `<miner address>`,
	Action: func(c *cli.Context) error {
		url, token := c.String("node"), c.String("token")

		if c.NArg() != 1 {
			return cli.ShowCommandHelp(c, "check")
		}

		mAddr, err := address.NewFromString(c.Args().First())
		if err != nil {
			return err
		}

		node, closer, err := NewRpcClient(url, &token)
		if err != nil {
			return err
		}
		defer closer()

		minerInfo, err := node.StateMinerInfo(c.Context, mAddr, types.EmptyTSK)
		if err != nil {
			return err
		}

		if minerInfo.PeerId == nil {
			return fmt.Errorf("miner(%s) peer id is nil", mAddr)
		}
		if len(minerInfo.Multiaddrs) == 0 {
			return fmt.Errorf("miner(%s) multiaddrs is empty", mAddr)
		}

		host, err := libp2p.New(libp2p.NoListenAddrs)
		if err != nil {
			return err
		}
		defer host.Close()

		addrInfo := peer.AddrInfo{
			ID:    *minerInfo.PeerId,
			Addrs: []multiaddr.Multiaddr{},
		}

		for _, addr := range minerInfo.Multiaddrs {
			maddr, err := multiaddr.NewMultiaddrBytes(addr)
			if err != nil {
				return fmt.Errorf("parsing multiaddr %s: %w", addr, err)
			}
			addrInfo.Addrs = append(addrInfo.Addrs, maddr)
		}

		if err := host.Connect(c.Context, addrInfo); err != nil {
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

		fmt.Println(userAgent)

		return nil
	},
}
