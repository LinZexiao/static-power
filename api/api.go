package api

import (
	"errors"
	"log"
	"strings"

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

func (a *Api) getMiner(id abi.ActorID) (*Miner, error) {
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

func (a *Api) getMiners(ids ...abi.ActorID) ([]Miner, error) {
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

func (a *Api) GetVenusStatic() (*StaticInfo, error) {
	venus_agent := []AgentInfo{}
	db.Where("name like ?", "%venus%").Or("name like ?", "%droplet%").Or("name like ?", "%market%").Find(&venus_agent)
	venus_power := []PowerInfo{}
	db.Where("miner_id in ?", sliceMap(venus_agent, func(a AgentInfo) abi.ActorID { return a.MinerID })).Find(&venus_power)
	return staticByPower(venus_power, true), nil
}

func (a *Api) GetLotusStatic() (*StaticInfo, error) {
	lotus_agent := []AgentInfo{}
	db.Where("name like ?", "%lotus%").Or("name like ?", "%boost%").Find(&lotus_agent)
	lotus_power := []PowerInfo{}
	db.Where("miner_id in ?", sliceMap(lotus_agent, func(a AgentInfo) abi.ActorID { return a.MinerID })).Find(&lotus_power)
	return staticByPower(lotus_power, true), nil
}

func (a *Api) GetProportion() (float64, error) {
	venus_agent := []AgentInfo{}
	lotus_agent := []AgentInfo{}
	// name contains venus or droplet or market
	db.Where("name like ?", "%venus%").Or("name like ?", "%droplet%").Or("name like ?", "%market%").Find(&venus_agent)
	// name contains lotus or boost
	db.Where("name like ?", "%lotus%").Or("name like ?", "%boost%").Find(&lotus_agent)

	venus_power := []PowerInfo{}
	lotus_power := []PowerInfo{}

	db.Where("miner_id in ?", sliceMap(venus_agent, func(a AgentInfo) abi.ActorID { return a.MinerID })).Find(&venus_power)
	db.Where("miner_id in ?", sliceMap(lotus_agent, func(a AgentInfo) abi.ActorID { return a.MinerID })).Find(&lotus_power)

	venus_static := staticByPower(venus_power, false)
	lotus_static := staticByPower(lotus_power, false)

	if venus_static.QAP == 0 {
		return 0.0, nil
	}
	return venus_static.QAP / (venus_static.QAP + lotus_static.QAP), nil
}

// get miner info to query agent
func (a *Api) GetAllMiners() ([]Miner, error) {
	var miners []Miner
	db.Find(&miners)
	return a.getMiners(sliceMap(miners, func(m Miner) abi.ActorID { return m.ID })...)
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

type StaticInfo struct {
	Count int
	RBP   float64
	QAP   float64

	// Raw Power of DCP sector
	DCP float64
	// Raw Power of CCP sector
	CCP float64
}

func staticByPower(powers []PowerInfo, excludeCcOnly bool) *StaticInfo {
	ret := StaticInfo{
		Count: len(powers),
	}
	for _, p := range powers {
		RBP := float64(p.RawBytePower.Uint64()) / PiB
		QAP := float64(p.QualityAdjPower.Uint64()) / PiB

		DCP := (QAP - RBP) / 9
		CCP := RBP - DCP

		ccOnly := CCP > 0.0000000001 && DCP < 0.0000000001
		if excludeCcOnly && ccOnly {
			log.Printf("miner(%d) has no DC power", p.MinerID)
			continue
		}

		ret.RBP += RBP
		ret.QAP += QAP
		ret.DCP += DCP
		ret.CCP += CCP
	}
	return &ret
}

func static(miners []Miner, excludeCcOnly bool) *StaticInfo {
	ret := StaticInfo{
		Count: len(miners),
	}
	for _, miner := range miners {
		if miner.Power == nil {
			log.Printf("miner(%d) has no power info", miner.ID)
			continue
		}
		power := *miner.Power
		RBP := float64(power.RawBytePower.Uint64()) / PiB
		QAP := float64(power.QualityAdjPower.Uint64()) / PiB

		DCP := (QAP - RBP) / 9
		CCP := RBP - DCP

		ccOnly := CCP > 0.0000000001 && DCP < 0.0000000001
		if excludeCcOnly && ccOnly {
			log.Printf("miner(%d) has no DC power", miner.ID)
			continue
		}

		ret.Count++
		ret.RBP += RBP
		ret.QAP += QAP
		ret.DCP += DCP
		ret.CCP += CCP
	}
	return &ret
}

func processData(miners []Miner) map[string]*StaticInfo {
	// data process

	var (
		All    = "All SP"
		Venus  = "Venus SP"
		Lotus  = "Lotus SP"
		Others = "Others SP"

		Deal      = "DC SP"
		LotusDeal = "Lotus DC SP"
		VenusDeal = "Venus DC SP"
		OtherDeal = "Other DC SP"
	)

	staticInfo := make(map[string]*StaticInfo)
	staticInfo[All] = &StaticInfo{}
	staticInfo[Venus] = &StaticInfo{}
	staticInfo[Lotus] = &StaticInfo{}
	staticInfo[Others] = &StaticInfo{}
	staticInfo[Deal] = &StaticInfo{}
	staticInfo[LotusDeal] = &StaticInfo{}
	staticInfo[VenusDeal] = &StaticInfo{}
	staticInfo[OtherDeal] = &StaticInfo{}

	for _, miner := range miners {
		if miner.Power == nil {
			log.Printf("miner(%d) has no power info", miner.ID)
			continue
		}
		if miner.Agent == nil {
			log.Printf("miner(%d) has no agent info", miner.ID)
			continue
		}
		power := *miner.Power
		agent := *miner.Agent
		RBP := float64(power.RawBytePower.Uint64()) / PiB
		QAP := float64(power.QualityAdjPower.Uint64()) / PiB

		DCP := (QAP - RBP) / 9
		CCP := RBP - DCP

		// then Condition  should be great than zero , but consider the influence of float64, so we use small enough value
		hasDeal := DCP > 0.0000000001

		update := func(name string) {
			a := staticInfo[name]
			a.Count++
			a.RBP += RBP
			a.QAP += QAP
			a.DCP += DCP
			a.CCP += CCP
			staticInfo[name] = a
		}

		update(All)
		if isVenus(agent.Name) {
			update(Venus)
		} else if isLotus(agent.Name) {
			update(Lotus)
		} else {
			update(Others)
		}

		if hasDeal {
			update(Deal)
			if isVenus(agent.Name) {
				update(VenusDeal)
			} else if isLotus(agent.Name) {
				update(LotusDeal)
			} else {
				update(OtherDeal)
			}
		}
	}

	return staticInfo
}

func isVenus(s string) bool {
	return strings.Contains(s, "venus") || strings.Contains(s, "droplet")
}

func isLotus(s string) bool {
	return strings.Contains(s, "lotus") || strings.Contains(s, "boost")
}

func sliceMap[T, U any](s []T, f func(T) U) []U {
	var ret []U
	for _, v := range s {
		ret = append(ret, f(v))
	}
	return ret
}
