package api

import (
	"fmt"
	"static-power/core"
	"testing"
	"time"

	"github.com/test-go/testify/require"
)

func TestFind(t *testing.T) {
	db := realDB(t, "/root/tanlang/others/static-power/test.db")
	a := NewApi(db)

	res, err := a.find(Option{Tag: "invalid_tag"})
	require.NoError(t, err)
	require.Len(t, res, 0)
}

func TestFindVenus(t *testing.T) {
	db := realDB(t, "/root/tanlang/others/static-power/test.db")
	a := NewApi(db)

	// bf := time.Now().Add(-time.Hour * 4)
	bf := time.Now()
	res, err := a.find(Option{Tag: "Japan", Before: bf, AgentType: core.AgentTypeVenus})
	require.NoError(t, err)
	fmt.Println(len(res), res)
}

func TestQueryStatement(t *testing.T) {
	db := realDB(t, "/root/tanlang/others/static-power/test.db")
	commonQuery := db.Order("updated_at asc")

	var power PowerInfo
	commonQuery.First(&power)
	fmt.Println(power)

	var miner Miner
	// minerQuery := commonQuery.Clone
	commonQuery.First(&miner)
	fmt.Println(miner)

	var peer PeerInfo
	commonQuery.First(&peer)
	fmt.Println(peer)

}

// SELECT * FROM `power_infos` ORDER BY updated_at desc,`power_infos`.`miner_id` LIMIT 1
// SELECT * FROM `power_infos` ORDER BY updated_at desc,`power_infos`.`miner_id`,`power_infos`.`miner_id` LIMIT 1
