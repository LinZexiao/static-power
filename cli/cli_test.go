package cli

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/test-go/testify/assert"
	"github.com/test-go/testify/require"
)

func TestRpcNode(t *testing.T) {
	ctx := context.Background()

	url := "ws://192.168.200.132:3453/rpc/v1"
	token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJuYW1lIjoiYmFpeXVfdmVudXNfdGVzdF9mb3JjZXNlYWxlciIsInBlcm0iOiJ3cml0ZSIsImV4dCI6IiJ9.OfwrhnK-qasTd3iLM50BL1b3vYgIBz5_NRVcA-FsaKw"

	node, closer, err := NewRpcClient(url, &token)
	assert.NoError(t, err)
	defer closer()

	miners, err := node.StateListMiners(ctx, types.EmptyTSK)
	assert.NoError(t, err)
	fmt.Println(miners)

	miner, err := address.NewFromString("f01036")
	assert.NoError(t, err)

	info, err := node.StateMinerInfo(ctx, miner, types.EmptyTSK)
	assert.NoError(t, err)
	fmt.Println(info)

	power, err := node.StateMinerPower(ctx, miner, types.EmptyTSK)
	require.NoError(t, err)
	fmt.Println(power, power.HasMinPower)

	var ids []uint64
	for _, m := range miners {
		id, err := address.IDFromAddress(m)
		assert.NoError(t, err)
		ids = append(ids, id)
		require.True(t, strings.HasSuffix(m.String(), fmt.Sprintf("%d", id)))
	}
}

func TestRpcNode2(t *testing.T) {

	url := "ws://192.168.200.132:3453/rpc/v1"
	token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJuYW1lIjoiYmFpeXVfdmVudXNfdGVzdF9mb3JjZXNlYWxlciIsInBlcm0iOiJ3cml0ZSIsImV4dCI6IiJ9.OfwrhnK-qasTd3iLM50BL1b3vYgIBz5_NRVcA-FsaKw"

	node, closer, err := NewRpcClient(url, &token)
	assert.NoError(t, err)
	defer closer()

	mis, err := getMinerInfosWithMinPower(node)
	require.NoError(t, err)
	fmt.Println(len(mis))
	for _, mi := range mis {
		fmt.Print(mi.ID, ",")
		if mi.Peer != nil {
			fmt.Print(*mi.Peer, ",")
			for _, addr := range *mi.Peer.Multiaddrs {
				fmt.Print(addr, ",")
			}
		}
		if mi.Power != nil {
			fmt.Print(*mi.Power, ",")
		}
		fmt.Println()
	}
	// err = statMinerPower(node)
	// assert.NoError(t, err)

}

func TestPeerConnect(t *testing.T) {
	err := connectPeer()
	require.NoError(t, err)
}

func connectPeer() error {
	ctx := context.Background()

	host, err := libp2p.New(libp2p.NoListenAddrs)
	if err != nil {
		return err
	}
	defer host.Close()

	// info := &api.PeerInfo{
	// 	PeerId: "12D3KooWJ7rb29A6TUvH7pwCvfJzCph2tXR7K6MR8mJJkUVCz4BN",
	// 	// Multiaddrs: &Multiaddrs{"test_addr1", "test_addr2"},
	// }

	peerId, err := peer.Decode("12D3KooWJ7rb29A6TUvH7pwCvfJzCph2tXR7K6MR8mJJkUVCz4BN")
	if err != nil {
		return err
	}

	ma, err := multiaddr.NewMultiaddr("/ip4/118.140.26.165/tcp/30001")
	if err != nil {
		return err
	}

	addrInfo := peer.AddrInfo{
		ID:    peerId,
		Addrs: []multiaddr.Multiaddr{ma},
	}

	// for _, addr := range info.Multiaddrs {
	// 	maddr, err := multiaddr.NewMultiaddr(addr)
	// 	if err != nil {
	// 		return fmt.Errorf("parsing multiaddr %s: %w", addr, err)
	// 	}
	// 	addrInfo.Addrs = append(addrInfo.Addrs, maddr)
	// }

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
	// info.Agent = userAgent

	fmt.Println(userAgent)

	return nil
}
