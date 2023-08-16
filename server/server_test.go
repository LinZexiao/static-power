package server

import (
	"static-power/api"
	"testing"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/test-go/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	return db
}

func TestHttp(t *testing.T) {
	db := newDB(t)
	a := api.NewApi(db)

	RegisterApi(a)
	Run()

	t.Run("get miners", func(t *testing.T) {
		miner := abi.ActorID(1002)

		m := &api.Multiaddrs{"test_addr1", "test_addr2"}
		peer0 := &api.PeerInfo{
			MinerID:    miner,
			PeerId:     "test_peer0",
			Multiaddrs: m,
		}
		peer1 := &api.PeerInfo{
			MinerID:    miner,
			PeerId:     "test_peer1",
			Multiaddrs: m,
		}

		err := UpdatePeerInfo(peer0)
		require.NoError(t, err)
		err = UpdatePeerInfo(peer1)
		require.NoError(t, err)

		agent0 := &api.AgentInfo{
			MinerID: miner,
			Name:    "test_agent_0",
		}
		agent1 := &api.AgentInfo{
			MinerID: miner,
			Name:    "test_agent_0",
		}

		err = UpdateAgentInfo(agent0)
		require.NoError(t, err)
		err = UpdateAgentInfo(agent1)
		require.NoError(t, err)

		p1000 := api.Power((big.NewInt(1000)))
		p2000 := api.Power((big.NewInt(2000)))
		power0 := &api.PowerInfo{
			MinerID:                  miner,
			RawBytePower:             &p1000,
			QualityAdjustedBytePower: &p1000,
		}
		power1 := &api.PowerInfo{
			MinerID:                  miner,
			RawBytePower:             &p2000,
			QualityAdjustedBytePower: &p2000,
		}

		err = UpdatePowerInfo(power0)
		require.NoError(t, err)
		err = UpdatePowerInfo(power1)
		require.NoError(t, err)

		res, err := GetMiners()
		require.NoError(t, err)
		require.Len(t, res, 1)
		require.Equal(t, peer1.PeerId, res[0].Peer.PeerId)
		require.Equal(t, agent1.Name, res[0].Agent.Name)
		require.Equal(t, power1.RawBytePower, res[0].Power.RawBytePower)
	})
}
