package api

import (
	"errors"
	"fmt"
	"log"
	"static-power/util"
	"strings"
	"time"

	"github.com/filecoin-project/go-state-types/abi"
	"gorm.io/gorm"
)

// could not be zero, because zero represent nil in gorm
var NetWork abi.ActorID = 1

var db *gorm.DB = nil

func NewApi(d *gorm.DB) *Api {
	d.AutoMigrate(&Miner{}, &PeerInfo{}, &PowerInfo{}, &AgentInfo{})
	db = d
	return &Api{}
}

func (a *Api) getMiner(id abi.ActorID, before ...time.Time) (*Miner, error) {
	var miner Miner
	err := db.First(&miner, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	var peer PeerInfo
	err = db.Order("updated_at desc").First(&peer, "miner_id = ?", miner.ID).Error
	if err == nil {
		miner.Peer = &peer
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	var power PowerInfo
	err = db.Order("updated_at desc").First(&power, "miner_id = ?", miner.ID).Error
	if err == nil {
		miner.Power = &power
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	var agent AgentInfo
	err = db.Order("updated_at desc").First(&agent, "miner_id = ?", miner.ID).Error
	if err == nil {
		miner.Agent = &agent
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	return &miner, nil
}

func (a *Api) getPowers(before time.Time, ids ...abi.ActorID) ([]PowerInfo, error) {
	ids = util.Unique(ids)

	var powers []PowerInfo
	// 获取所有 miner_id in (ids) 的最新的 power 信息
	query := db.Select("miner_id, raw_byte_power ,quality_adj_power, updated_at,  max(updated_at) as max_updated_at").Where("miner_id in ?", ids)
	if !before.IsZero() {
		query = query.Where("updated_at < ?", before)
	}

	err := query.Group("miner_id").Table("power_infos").Find(&powers).Error

	// err := db.Joins("inner join (?) as subquery on power_infos.miner_id = subquery.miner_id and power_infos.updated_at = subquery.updated_at", subquery).Find(&powers, "miner_id in ?", ids).Error
	if err != nil {
		return nil, err
	}
	return powers, nil
}

func (a *Api) getAgents(ids ...abi.ActorID) ([]AgentInfo, error) {
	ids = util.Unique(ids)

	var agents []AgentInfo
	err := db.Select("miner_id, name, updated_at,  max(updated_at) as max_updated_at").Where("miner_id in ?", ids).Group("miner_id").Table("agent_infos").Find(&agents).Error
	if err != nil {
		return nil, err
	}
	return agents, nil
}

func (a *Api) getMiners(ids ...abi.ActorID) ([]Miner, error) {
	ids = util.Unique(ids)

	var miners []Miner
	for _, id := range ids {
		miner, err := a.getMiner(id)
		if err != nil {
			return nil, err
		}
		miners = append(miners, *miner)
	}
	return miners, nil
}

func (a *Api) find(opt Option) ([]abi.ActorID, error) {
	agent := []AgentInfo{}

	// name contains venus or droplet or market
	subQuery := db.Select("miner_id, name, tag, updated_at,  max(updated_at) as max_updated_at")
	if !opt.Before.IsZero() {

		// 避免 查询时间戳落入 查询更新时间段
		var latestUpdatedAt time.Time
		timeQuery := db.Table("agent_infos").Select("max(updated_at) as max_updated_at").Where("updated_at < ?", opt.Before)
		if opt.Tag != "" {
			timeQuery = timeQuery.Where("tag = ?", opt.Tag)
		}
		err := timeQuery.Scan(&latestUpdatedAt).Error
		if err != nil && !strings.Contains(err.Error(), "unsupported Scan") {
			return nil, fmt.Errorf("get latest update time : %w", err)
		}
		if opt.Before.Sub(latestUpdatedAt) < 5*time.Minute {
			log.Printf("latest update time is too close to query time, query: %s, latest: %s", opt.Before, latestUpdatedAt)
			opt.Before = opt.Before.Add(5 * time.Minute)
		}
		subQuery = subQuery.Where("updated_at < ?", opt.Before)
	}
	if opt.Tag != "" {
		subQuery = subQuery.Where("tag = ?", opt.Tag)
	}
	subQuery = subQuery.Group("miner_id").Table("agent_infos")

	err := subQuery.Find(&agent).Error
	if err != nil {
		return nil, err
	}

	var maxUpdatedAt time.Time
	var minUpdatedAt = time.Now()

	for _, agent := range agent {
		if agent.UpdatedAt.After(maxUpdatedAt) {
			maxUpdatedAt = agent.UpdatedAt
		}
		if agent.UpdatedAt.Before(minUpdatedAt) {
			minUpdatedAt = agent.UpdatedAt
		}
	}

	// rm agent which updated_at is not too old
	var tmp []AgentInfo
	for _, agent := range agent {
		if agent.UpdatedAt.After(maxUpdatedAt.Add(-70 * time.Minute)) {
			tmp = append(tmp, agent)
		}
	}

	agent = tmp

	ids := util.SliceMap(agent, func(a AgentInfo) abi.ActorID { return a.MinerID })
	ids = util.Unique(ids)

	return ids, nil
}

func (a *Api) findVenus(opt Option) ([]abi.ActorID, error) {
	venus_agent := []AgentInfo{}

	// name contains venus or droplet or market
	subQuery := db.Select("miner_id, name, tag, updated_at,  max(updated_at) as max_updated_at")
	if !opt.Before.IsZero() {

		// 避免 查询时间戳落入 查询更新时间段
		var latestUpdatedAt time.Time
		timeQuery := db.Table("agent_infos").Select("max(updated_at) as max_updated_at").Where("updated_at < ?", opt.Before)
		if opt.Tag != "" {
			timeQuery = timeQuery.Where("tag = ?", opt.Tag)
		}
		err := timeQuery.Scan(&latestUpdatedAt).Error
		if err != nil && !strings.Contains(err.Error(), "unsupported Scan") {
			return nil, fmt.Errorf("get latest update time : %w", err)
		}
		if opt.Before.Sub(latestUpdatedAt) < 5*time.Minute {
			log.Printf("latest update time is too close to query time, query: %s, latest: %s", opt.Before, latestUpdatedAt)
			opt.Before = opt.Before.Add(5 * time.Minute)
		}

		subQuery = subQuery.Where("updated_at < ?", opt.Before)
	}
	if opt.Tag != "" {
		subQuery = subQuery.Where("tag = ?", opt.Tag)
	}
	subQuery = subQuery.Group("miner_id").Table("agent_infos")
	err := db.Table("(?) as t", subQuery).Where("name like ?", "%venus%").Or("name like ?", "%droplet%").Or("name like ?", "%market%").Find(&venus_agent).Error
	if err != nil {
		return nil, err
	}

	var maxUpdatedAt time.Time
	var minUpdatedAt = time.Now()

	for _, agent := range venus_agent {
		if agent.UpdatedAt.After(maxUpdatedAt) {
			maxUpdatedAt = agent.UpdatedAt
		}
		if agent.UpdatedAt.Before(minUpdatedAt) {
			minUpdatedAt = agent.UpdatedAt
		}
	}

	// rm agent which updated_at is not too old
	var tmp []AgentInfo
	for _, agent := range venus_agent {
		if agent.UpdatedAt.After(maxUpdatedAt.Add(-10 * time.Minute)) {
			tmp = append(tmp, agent)
		}
	}

	venus_agent = tmp

	ids := util.SliceMap(venus_agent, func(a AgentInfo) abi.ActorID { return a.MinerID })
	ids = util.Unique(ids)

	return ids, nil
}

func (a *Api) findLotus(opt Option) ([]abi.ActorID, error) {
	lotus_agent := []AgentInfo{}

	// name contains lotus or boost
	subQuery := db.Select("miner_id, name, tag , updated_at,  max(updated_at) as max_updated_at")
	if !opt.Before.IsZero() {

		// 避免 查询时间戳落入 查询更新时间段
		var latestUpdatedAt time.Time
		timeQuery := db.Table("agent_infos").Select("max(updated_at) as max_updated_at").Where("updated_at < ?", opt.Before)
		if opt.Tag != "" {
			timeQuery = timeQuery.Where("tag = ?", opt.Tag)
		}
		err := timeQuery.Scan(&latestUpdatedAt).Error
		if err != nil && !strings.Contains(err.Error(), "unsupported Scan") {
			return nil, fmt.Errorf("get latest update time : %w", err)
		}
		if opt.Before.Sub(latestUpdatedAt) < 5*time.Minute {
			log.Printf("latest update time is too close to query time, query: %s, latest: %s", opt.Before, latestUpdatedAt)
			opt.Before = opt.Before.Add(5 * time.Minute)
		}

		subQuery = subQuery.Where("updated_at < ?", opt.Before)
	}
	if opt.Tag != "" {
		subQuery = subQuery.Where("tag = ?", opt.Tag)
	}
	subQuery = subQuery.Group("miner_id").Table("agent_infos")
	err := db.Table("(?) as t", subQuery).Where("name like ?", "%lotus%").Or("name like ?", "%boost%").Find(&lotus_agent).Error
	if err != nil {
		return nil, err
	}

	var maxUpdatedAt time.Time
	var minUpdatedAt = time.Now()

	for _, agent := range lotus_agent {
		if agent.UpdatedAt.After(maxUpdatedAt) {
			maxUpdatedAt = agent.UpdatedAt
		}
		if agent.UpdatedAt.Before(minUpdatedAt) {
			minUpdatedAt = agent.UpdatedAt
		}
	}

	// rm agent which updated_at is not too old
	var tmp []AgentInfo
	for _, agent := range lotus_agent {
		if agent.UpdatedAt.After(maxUpdatedAt.Add(-10 * time.Minute)) {
			tmp = append(tmp, agent)
		}
	}

	lotus_agent = tmp

	ids := util.SliceMap(lotus_agent, func(a AgentInfo) abi.ActorID { return a.MinerID })
	ids = util.Unique(ids)

	return ids, nil
}

func (a *Api) GetVenusStatic(opt Option) (*StaticInfo, error) {
	venusId, err := a.findVenus(opt)
	if err != nil {
		return nil, err
	}
	venus_power, err := a.getPowers(opt.Before, venusId...)
	if err != nil {
		return nil, err
	}
	return staticByPower(venus_power, false), nil
}

func (a *Api) GetLotusStatic(opt Option) (*StaticInfo, error) {
	lotusId, err := a.findLotus(opt)
	if err != nil {
		return nil, err
	}
	lotus_power, err := a.getPowers(opt.Before, lotusId...)
	if err != nil {
		return nil, err
	}
	return staticByPower(lotus_power, false), nil
}

func (a *Api) GetProportion(opt Option) (float64, error) {
	venusStaticInfo, err := a.GetVenusStatic(opt)
	if err != nil {
		return 0.0, err
	}
	lotusStaticINfo, err := a.GetLotusStatic(opt)
	if err != nil {
		return 0.0, err
	}

	if venusStaticInfo.QAP == 0 {
		return 0.0, nil
	}
	log.Printf("venus_static.QAP: %f, lotus_static.QAP: %f", venusStaticInfo.QAP, lotusStaticINfo.QAP)
	return venusStaticInfo.QAP / (venusStaticInfo.QAP + lotusStaticINfo.QAP), nil
}

// get miner info to query agent
func (a *Api) GetAllMiners() ([]Miner, error) {
	var miners []Miner
	db.Find(&miners)
	return a.getMiners(util.SliceMap(miners, func(m Miner) abi.ActorID { return m.ID })...)
}

// export miners
func (a *Api) GetMiners(opt Option) ([]Miner, error) {

	minerIDs, err := a.find(opt)
	if err != nil {
		return nil, err
	}

	return a.getMiners(minerIDs...)
}

// update miner Agent
func (a *Api) UpdateMinerAgentInfo(agent *AgentInfo) error {
	err := db.Save(&Miner{ID: agent.MinerID}).Error
	if err != nil {
		return err
	}
	err = db.Create(agent).Error
	if err != nil {
		return err
	}
	return nil
}

// update Miner PeerInfo
func (a *Api) UpdateMinerPeerInfo(peer *PeerInfo) error {
	err := db.Save(&Miner{ID: peer.MinerID}).Error
	if err != nil {
		return err
	}
	err = db.Create(peer).Error
	if err != nil {
		return err
	}
	return nil
}

// update Miner PowerInfo
func (a *Api) UpdateMinerPowerInfo(power *PowerInfo) error {
	err := db.Save(&Miner{ID: power.MinerID}).Error
	if err != nil {
		return err
	}
	err = db.Create(power).Error
	if err != nil {
		return err
	}
	return nil
}

const PiB float64 = 1024 * 1024 * 1024 * 1024 * 1024

func CleanUp(before time.Time) {
	// insert state before delete
	db.Where("updated_at < ?", before).Delete(&PowerInfo{})
	db.Where("updated_at < ?", before).Delete(&AgentInfo{})
	db.Where("updated_at < ?", before).Delete(&PeerInfo{})
}
