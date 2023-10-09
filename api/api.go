package api

import (
	"errors"
	"log"
	"static-power/core"
	"static-power/util"
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

func (a *Api) getMiner(id abi.ActorID, opts ...Option) (*Miner, error) {
	opt := Option{}
	if len(opts) > 0 {
		opt = opts[0]
	}

	var miner Miner
	err := db.First(&miner, "id = ?", id).Error
	if err != nil {
		return nil, err
	}

	commonScope := func(db *gorm.DB) *gorm.DB {
		ret := db.Order("updated_at desc").Where("miner_id = ?", id)
		if !opt.Before.IsZero() {
			ret = ret.Where("updated_at < ?", opt.Before)
		}
		return ret
	}

	specifyTag := func(db *gorm.DB) *gorm.DB {
		if opt.Tag != "" {
			return db.Where("tag = ?", opt.Tag)
		}
		return db
	}

	var peer PeerInfo
	err = db.Scopes(commonScope).First(&peer).Error
	if err == nil {
		miner.Peer = &peer
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	var power PowerInfo
	err = db.Scopes(commonScope).First(&power).Error
	if err == nil {
		miner.Power = &power
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	// some miner may has no power when it was punished
	if miner.Power != nil {
		if power.QualityAdjPower == nil || power.RawBytePower == nil {
			log.Printf("miner(%d) has no power when(%v)", id, opt)
		}
	}

	var agent AgentInfo
	err = db.Scopes(commonScope, specifyTag).First(&agent).Error
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

	if err != nil {
		return nil, err
	}
	return powers, nil
}

func (a *Api) getMiners(opt Option, ids ...abi.ActorID) ([]Miner, error) {
	ids = util.Unique(ids)

	var miners []Miner
	for _, id := range ids {
		miner, err := a.getMiner(id, opt)
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
		// var latestUpdatedAt time.Time
		// timeQuery := db.Table("agent_infos").Select("max(updated_at) as max_updated_at").Where("updated_at < ?", opt.Before)
		// if opt.Tag != "" {
		// 	timeQuery = timeQuery.Where("tag = ?", opt.Tag)
		// }
		// err := timeQuery.Scan(&latestUpdatedAt).Error
		// if err != nil && !strings.Contains(err.Error(), "unsupported Scan") {
		// 	return nil, fmt.Errorf("get latest update time : %w", err)
		// }
		// if opt.Before.Sub(latestUpdatedAt) < 5*time.Minute {
		// 	log.Printf("latest update time is too close to query time, query: %s, latest: %s", opt.Before, latestUpdatedAt)
		// 	opt.Before = opt.Before.Add(5 * time.Minute)
		// }

		subQuery = subQuery.Where("updated_at < ?", opt.Before)
	}
	if opt.Tag != "" {
		subQuery = subQuery.Where("tag = ?", opt.Tag)
	}
	subQuery = subQuery.Group("miner_id").Table("agent_infos")

	var err error
	switch opt.AgentType {
	case core.AgentTypeOther:
		err = subQuery.Find(&agent).Error
	case core.AgentTypeLotus:
		err = db.Table("(?) as t", subQuery).Where("name like ?", "%lotus%").Or("name like ?", "%boost%").Find(&agent).Error
	case core.AgentTypeVenus:
		err = db.Table("(?) as t", subQuery).Where("name like ?", "%venus%").Or("name like ?", "%droplet%").Or("name like ?", "%market%").Find(&agent).Error
	}
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

func (a *Api) GetStatic(opt Option) (*StaticInfo, error) {
	venusId, err := a.find(opt)
	if err != nil {
		return nil, err
	}

	venus_power, err := a.getPowers(opt.Before, venusId...)
	if err != nil {
		return nil, err
	}
	return staticByPower(venus_power, false), nil
}

func (a *Api) GetProportion(opt Option) (float64, error) {
	opt.AgentType = core.AgentTypeVenus
	venusStaticInfo, err := a.GetStatic(opt)
	if err != nil {
		return 0.0, err
	}

	opt.AgentType = core.AgentTypeLotus
	lotusStaticINfo, err := a.GetStatic(opt)
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
	return a.getMiners(Option{}, util.SliceMap(miners, func(m Miner) abi.ActorID { return m.ID })...)
}

// export miners
func (a *Api) GetMiners(opt Option) ([]Miner, error) {
	minerIDs, err := a.find(opt)
	if err != nil {
		return nil, err
	}

	return a.getMiners(opt, minerIDs...)
}

// diff QAP miners
func (a *Api) Diff(opt Option) ([]map[core.AgentType]core.Summary, []core.Difference, error) {

	before, err := a.GetMiners(opt)
	if err != nil {
		return nil, nil, err
	}
	opt.Before = opt.After
	after, err := a.GetMiners(Option{Before: opt.Before.Add(1 * time.Minute)})

	briefBefore := util.SliceMap(before, GetBrief)
	briefAfter := util.SliceMap(after, GetBrief)

	difference := core.Diff(briefBefore, briefAfter)

	summaryBefore := core.Summarize(briefBefore)
	summaryAfter := core.Summarize(briefAfter)

	summaries := []map[core.AgentType]core.Summary{summaryBefore, summaryAfter}

	return summaries, difference, nil
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
