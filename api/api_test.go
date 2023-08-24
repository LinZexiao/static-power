package api

import (
	"encoding/json"
	"fmt"
	mbig "math/big"
	"testing"
	"time"

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

		res, err := api.GetAllMiners()
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
			MinerID:         miner,
			RawBytePower:    &p,
			QualityAdjPower: &p,
		}

		err = api.UpdateMinerPowerInfo(power)
		require.NoError(t, err)

		res, err := api.GetAllMiners()
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

		res, err := api.GetAllMiners()
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
			MinerID:         miner,
			RawBytePower:    &p1000,
			QualityAdjPower: &p1000,
		}

		power1 := &PowerInfo{
			MinerID:         miner,
			RawBytePower:    &p2000,
			QualityAdjPower: &p2000,
		}

		err = api.UpdateMinerPowerInfo(power0)
		require.NoError(t, err)

		err = api.UpdateMinerPowerInfo(power1)
		require.NoError(t, err)

		res, err := api.GetAllMiners()
		require.NoError(t, err)
		require.Len(t, res, 1)
		require.Equal(t, peer1.PeerId, res[0].Peer.PeerId)
		require.Equal(t, agent1.Name, res[0].Agent.Name)
		require.Equal(t, power1.RawBytePower, res[0].Power.RawBytePower)

	})

	t.Run("update power for network", func(t *testing.T) {
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		require.NoError(t, err)

		api := NewApi(db)

		p := Power((big.NewInt(1000)))
		power := &PowerInfo{
			MinerID:         NetWork,
			RawBytePower:    &p,
			QualityAdjPower: &p,
		}

		err = api.UpdateMinerPowerInfo(power)
		require.NoError(t, err)

		res, err := api.GetAllMiners()
		require.NoError(t, err)
		require.Equal(t, 1, len(res))
		power.UpdatedAt = res[0].Power.UpdatedAt
		require.Equal(t, *power, *res[0].Power)
	})
}

func TestStatic(t *testing.T) {
	t.Run("get miner info", func(t *testing.T) {
		db := newDB(t)

		api := NewApi(db)

		agents := []AgentInfo{
			{
				MinerID: abi.ActorID(1001),
				Name:    "dropletv",
			}, {
				MinerID: abi.ActorID(1002),
				Name:    "market_",
			}, {
				MinerID: abi.ActorID(1003),
				Name:    "_venus ",
			}, {
				MinerID: abi.ActorID(1004),
				Name:    " lotus ",
			}, {
				MinerID: abi.ActorID(1005),
				Name:    "boost v",
			},
		}

		for _, agent := range agents {
			err := api.UpdateMinerAgentInfo(&agent)
			require.NoError(t, err)
		}

		powers := []PowerInfo{
			{
				MinerID:         abi.ActorID(1001),
				RawBytePower:    pib(1),
				QualityAdjPower: pib(1),
			}, {
				MinerID:         abi.ActorID(1002),
				RawBytePower:    pib(2),
				QualityAdjPower: pib(2),
			},
			{
				MinerID:         abi.ActorID(1003),
				RawBytePower:    pib(3),
				QualityAdjPower: pib(3),
			},
			{
				MinerID:         abi.ActorID(1004),
				RawBytePower:    pib(4),
				QualityAdjPower: pib(4),
			},
			{
				MinerID:         abi.ActorID(1005),
				RawBytePower:    pib(5),
				QualityAdjPower: pib(5),
			},
		}

		for _, power := range powers {
			err := api.UpdateMinerPowerInfo(&power)
			require.NoError(t, err)
		}

		res, err := api.GetProportion(Option{})
		require.NoError(t, err)
		require.Equal(t, 0.4, res)
	})

	t.Run("get power info", func(t *testing.T) {
		db := newDB(t)

		api := NewApi(db)

		powersBeforeStamp := []PowerInfo{
			{
				MinerID:         abi.ActorID(1002),
				RawBytePower:    pib(1),
				QualityAdjPower: pib(1),
			}, {
				MinerID:         abi.ActorID(1002),
				RawBytePower:    pib(2),
				QualityAdjPower: pib(2),
			},
			{
				MinerID:         abi.ActorID(1003),
				RawBytePower:    pib(3),
				QualityAdjPower: pib(3),
			},
			{
				MinerID:         abi.ActorID(1005),
				RawBytePower:    pib(4),
				QualityAdjPower: pib(4),
			},
			{
				MinerID:         abi.ActorID(1005),
				RawBytePower:    pib(5),
				QualityAdjPower: pib(5),
			},
		}

		for _, power := range powersBeforeStamp {
			err := api.UpdateMinerPowerInfo(&power)
			require.NoError(t, err)
		}

		stamp := time.Now()
		powersAfterStamp := []PowerInfo{
			{
				MinerID:         abi.ActorID(1002),
				RawBytePower:    pib(5),
				QualityAdjPower: pib(5),
			}, {
				MinerID:         abi.ActorID(1002),
				RawBytePower:    pib(4),
				QualityAdjPower: pib(4),
			},
			{
				MinerID:         abi.ActorID(1005),
				RawBytePower:    pib(1),
				QualityAdjPower: pib(1),
			},
		}

		for _, power := range powersAfterStamp {
			err := api.UpdateMinerPowerInfo(&power)
			require.NoError(t, err)
		}

		res, err := api.getPowers(stamp, abi.ActorID(1002), abi.ActorID(1003), abi.ActorID(1005))
		require.NoError(t, err)
		for _, power := range res {
			switch power.MinerID {
			case abi.ActorID(1002):
				require.Equal(t, pib(2), power.RawBytePower)
				require.Equal(t, pib(2), power.QualityAdjPower)
			case abi.ActorID(1003):
				require.Equal(t, pib(3), power.RawBytePower)
				require.Equal(t, pib(3), power.QualityAdjPower)
			case abi.ActorID(1005):
				require.Equal(t, pib(5), power.RawBytePower)
				require.Equal(t, pib(5), power.QualityAdjPower)
			}
		}

		res, err = api.getPowers(time.Now(), abi.ActorID(1002), abi.ActorID(1003), abi.ActorID(1005))
		require.NoError(t, err)
		for _, power := range res {
			switch power.MinerID {
			case abi.ActorID(1002):
				require.Equal(t, pib(4), power.RawBytePower)
				require.Equal(t, pib(4), power.QualityAdjPower)
			case abi.ActorID(1003):
				require.Equal(t, pib(3), power.RawBytePower)
				require.Equal(t, pib(3), power.QualityAdjPower)
			case abi.ActorID(1005):
				require.Equal(t, pib(1), power.RawBytePower)
				require.Equal(t, pib(1), power.QualityAdjPower)
			}
		}

	})

	t.Run("get agent info", func(t *testing.T) {
		db := newDB(t)

		api := NewApi(db)

		agents := []AgentInfo{
			{
				MinerID: abi.ActorID(1001),
				Name:    "dropletv",
			}, {
				MinerID: abi.ActorID(1001),
				Name:    "lotus",
			}, {
				MinerID: abi.ActorID(1001),
				Name:    "venus_latest",
			}, {
				MinerID: abi.ActorID(1005),
				Name:    " market ",
			}, {
				MinerID: abi.ActorID(1005),
				Name:    "boost_latest",
			},
		}

		for _, agent := range agents {
			err := api.UpdateMinerAgentInfo(&agent)
			require.NoError(t, err)
		}

		res, err := api.getAgents(abi.ActorID(1001), abi.ActorID(1005))
		require.NoError(t, err)
		for _, agent := range res {
			switch agent.MinerID {
			case abi.ActorID(1001):
				require.Equal(t, "venus_latest", agent.Name)
			case abi.ActorID(1005):
				require.Equal(t, "boost_latest", agent.Name)
			}
		}
	})

	t.Run("find agent", func(t *testing.T) {
		db := newDB(t)

		api := NewApi(db)

		venusAgentName := "venus_std"
		lotusAgentName := "lotus_std"

		agents1 := []AgentInfo{
			{
				MinerID: abi.ActorID(1001),
				Name:    "dropletv",
			}, {
				MinerID: abi.ActorID(1001),
				Name:    "lotus",
			}, {
				MinerID: abi.ActorID(1001),
				Name:    venusAgentName,
			}, {
				MinerID: abi.ActorID(1005),
				Name:    " market ",
			}, {
				MinerID: abi.ActorID(1005),
				Name:    lotusAgentName,
			},
		}

		for _, agent := range agents1 {
			err := api.UpdateMinerAgentInfo(&agent)
			require.NoError(t, err)
		}

		stamp := time.Now()

		agents2 := []AgentInfo{
			{
				MinerID: abi.ActorID(1001),
				Name:    lotusAgentName,
			}, {
				MinerID: abi.ActorID(1005),
				Name:    venusAgentName,
			},
		}

		for _, agent := range agents2 {
			err := api.UpdateMinerAgentInfo(&agent)
			require.NoError(t, err)
		}

		// before stamp
		res, err := api.findVenus(Option{Before: stamp})
		require.NoError(t, err)
		require.Len(t, res, 1)
		require.Equal(t, abi.ActorID(1001), res[0])

		res, err = api.findLotus(Option{Before: stamp})
		require.NoError(t, err)
		require.Len(t, res, 1)
		require.Equal(t, abi.ActorID(1005), res[0])

		// before now
		res, err = api.findVenus(Option{})
		require.NoError(t, err)
		require.Len(t, res, 1)
		require.Equal(t, abi.ActorID(1005), res[0])

		res, err = api.findLotus(Option{})
		require.NoError(t, err)
		require.Len(t, res, 1)
		require.Equal(t, abi.ActorID(1001), res[0])
	})

	t.Run("find agent with tag", func(t *testing.T) {
		db := newDB(t)

		api := NewApi(db)

		venusAgentName := "venus_std"
		lotusAgentName := "lotus_std"

		agents1 := []AgentInfo{
			{
				MinerID: abi.ActorID(1001),
				Name:    "droplet",
			}, {
				MinerID: abi.ActorID(1001),
				Name:    "lotus",
			}, {
				MinerID: abi.ActorID(1001),
				Name:    venusAgentName,
			}, {
				MinerID: abi.ActorID(1005),
				Name:    " market ",
			}, {
				MinerID: abi.ActorID(1005),
				Name:    lotusAgentName,
			},
		}

		for _, agent := range agents1 {
			err := api.UpdateMinerAgentInfo(&agent)
			require.NoError(t, err)
		}

		// before now
		res, err := api.findVenus(Option{Tag: "HongKong"})
		require.NoError(t, err)
		require.Len(t, res, 0)

		res, err = api.findLotus(Option{Tag: "HongKong"})
		require.NoError(t, err)
		require.Len(t, res, 0)

		agents2 := []AgentInfo{
			{
				MinerID: abi.ActorID(1001),
				Name:    "droplet",
				Tag:     "HongKong",
			}, {
				MinerID: abi.ActorID(1005),
				Name:    "boost",
				Tag:     "HongKong",
			},
		}

		for _, agent := range agents2 {
			err := api.UpdateMinerAgentInfo(&agent)
			require.NoError(t, err)
		}

		res, err = api.findVenus(Option{Tag: "HongKong"})
		require.NoError(t, err)
		require.Len(t, res, 1)

		res, err = api.findLotus(Option{Tag: "HongKong"})
		require.NoError(t, err)
		require.Len(t, res, 1)

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
			MinerID:         1002,
			RawBytePower:    &p,
			QualityAdjPower: &p,
		}
		data, err := json.Marshal(power)
		require.NoError(t, err)
		fmt.Println(string(data))
	})

}

func newDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:?parseTime=true"), &gorm.Config{})
	require.NoError(t, err)
	return db
}

func pib(count int) *Power {
	p := Power((big.NewInt(int64(float64(count) * PiB))))
	return &p
}
