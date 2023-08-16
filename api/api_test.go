package api

import (
	"encoding/json"
	"fmt"
	mbig "math/big"
	"testing"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/test-go/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestApiUpdate(t *testing.T) {

	t.Run("update agent", func(t *testing.T) {
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		require.NoError(t, err)

		miner := abi.ActorID(1002)

		api := NewApi(db)
		agent := &AgentInfo{
			MinerID: miner,
			Name:    "test_agent",
		}

		err = api.UpdateMinerAgentInfo(agent)
		require.NoError(t, err)

		res, err := api.GetMinerInfo()
		require.NoError(t, err)
		require.Equal(t, 1, len(res))
		agent.UpdatedAt = res[0].Agent.UpdatedAt
		require.Equal(t, *agent, *res[0].Agent)
	})

	t.Run("update power", func(t *testing.T) {
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		require.NoError(t, err)

		miner := abi.ActorID(1002)

		api := NewApi(db)

		p := Power((big.NewInt(1000)))
		power := &PowerInfo{
			MinerID:                  miner,
			RawBytePower:             &p,
			QualityAdjustedBytePower: &p,
		}

		err = api.UpdateMinerPowerInfo(power)
		require.NoError(t, err)

		res, err := api.GetMinerInfo()
		require.NoError(t, err)
		require.Equal(t, 1, len(res))
		power.UpdatedAt = res[0].Power.UpdatedAt
		require.Equal(t, *power, *res[0].Power)
	})

	t.Run("update peer info", func(t *testing.T) {
		db := newDB(t)

		api := NewApi(db)
		miner := abi.ActorID(1002)

		m := Multiaddrs{"test_addr1", "test_addr2"}
		peer := &PeerInfo{
			MinerID:    miner,
			PeerId:     "test_peer",
			Multiaddrs: &m,
		}

		err := api.UpdateMinerPeerInfo(peer)
		require.NoError(t, err)

		res, err := api.GetMinerInfo()
		require.NoError(t, err)
		require.Len(t, res, 1)
		peer.UpdatedAt = res[0].Peer.UpdatedAt
		require.Equal(t, *peer, *res[0].Peer)
	})
}

func TestApiGetInfo(t *testing.T) {
	t.Run("get miner info", func(t *testing.T) {
		db := newDB(t)

		api := NewApi(db)
		miner := abi.ActorID(1002)

		m := &Multiaddrs{"test_addr1", "test_addr2"}
		peer0 := &PeerInfo{
			MinerID:    miner,
			PeerId:     "test_peer0",
			Multiaddrs: m,
		}

		peer1 := &PeerInfo{
			MinerID:    miner,
			PeerId:     "test_peer1",
			Multiaddrs: m,
		}

		err := api.UpdateMinerPeerInfo(peer0)
		require.NoError(t, err)

		err = api.UpdateMinerPeerInfo(peer1)
		require.NoError(t, err)

		agent0 := &AgentInfo{
			MinerID: miner,
			Name:    "test_agent_0",
		}

		agent1 := &AgentInfo{
			MinerID: miner,
			Name:    "test_agent_0",
		}

		err = api.UpdateMinerAgentInfo(agent0)
		require.NoError(t, err)
		err = api.UpdateMinerAgentInfo(agent1)
		require.NoError(t, err)

		p1000 := Power((big.NewInt(1000)))
		p2000 := Power((big.NewInt(2000)))

		power0 := &PowerInfo{
			MinerID:                  miner,
			RawBytePower:             &p1000,
			QualityAdjustedBytePower: &p1000,
		}

		power1 := &PowerInfo{
			MinerID:                  miner,
			RawBytePower:             &p2000,
			QualityAdjustedBytePower: &p2000,
		}

		err = api.UpdateMinerPowerInfo(power0)
		require.NoError(t, err)

		err = api.UpdateMinerPowerInfo(power1)
		require.NoError(t, err)

		res, err := api.GetMinerInfo()
		require.NoError(t, err)
		require.Len(t, res, 1)
		require.Equal(t, peer1.PeerId, res[0].Peer.PeerId)
		require.Equal(t, agent1.Name, res[0].Agent.Name)
		require.Equal(t, power1.RawBytePower, res[0].Power.RawBytePower)

	})
}

func TestJasonMarshal(t *testing.T) {

	t.Run("marshal math big", func(t *testing.T) {
		b := mbig.NewInt(1000)
		data, err := json.Marshal(b)
		require.NoError(t, err)
		fmt.Println(string(data))

		var b1 mbig.Int
		err = json.Unmarshal(data, &b1)
		require.NoError(t, err)
		require.Equal(t, b, &b1)
	})

	t.Run("marshal power", func(t *testing.T) {
		p := Power((big.NewInt(1000)))
		data, err := json.Marshal(p)
		require.NoError(t, err)
		fmt.Println(string(data))

		var p1 Power
		err = json.Unmarshal(data, &p1)
		require.NoError(t, err)
		require.Equal(t, p, p1)
	})

	t.Run("marshal power info ", func(t *testing.T) {
		p := Power((big.NewInt(1000)))
		power := &PowerInfo{
			MinerID:                  1002,
			RawBytePower:             &p,
			QualityAdjustedBytePower: &p,
		}
		data, err := json.Marshal(power)
		require.NoError(t, err)
		fmt.Println(string(data))
	})

}

func newDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	return db
}
