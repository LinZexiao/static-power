package main

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/test-go/testify/assert"
)

func TestRpcNode(t *testing.T) {
	ctx := context.Background()
	node, closer, err := newRpcClient("ws://192.168.200.132:3453/v1")
	assert.NoError(t, err)
	defer closer()

	miner, err := address.NewFromString("f01000")
	assert.NoError(t, err)

	info, err := node.StateMinerInfo(ctx, miner, types.EmptyTSK)
	assert.NoError(t, err)
	fmt.Println(info)

	// err = statMinerPower(node)
	// assert.NoError(t, err)

}

func newRpcClient(endpoint string) (api.FullNode, jsonrpc.ClientCloser, error) {
	requestHeader := http.Header{}
	var res api.FullNodeStruct
	closer, err := jsonrpc.NewMergeClient(context.Background(), endpoint, "Filecoin", api.GetInternalStructs(&res), requestHeader)
	return &res, closer, err
}
