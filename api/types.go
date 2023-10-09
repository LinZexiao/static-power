package api

import (
	"database/sql/driver"
	"errors"
	"log"
	"static-power/core"
	"static-power/util"
	"strings"
	"time"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
)

type Option struct {
	Before time.Time
	Tag    string

	// only use for diff with before
	After time.Time

	AgentType core.AgentType
}

type Power big.Int

const (
	TagLocationSingapore = "Singapore"
	TagLocationHongKong  = "HongKong"
	TagLocationJapan     = "Japan"
)

func (p *Power) Value() (driver.Value, error) {
	return p.String(), nil
}

func (p *Power) Scan(src interface{}) error {
	switch src := src.(type) {
	case string:
		s := string(src)
		res, err := big.FromString(s)
		if err != nil {
			return err
		}
		temp := (Power)(res)
		*p = temp
	default:
		return errors.New("invalid power")
	}
	return nil
}

func (p *Power) UnmarshalJSON(b []byte) error {
	res, err := big.FromString(string(b))
	if err != nil {
		return err
	}
	temp := (Power)(res)
	*p = temp
	return nil
}

type Multiaddrs []string

func (m *Multiaddrs) Value() (driver.Value, error) {
	s := strings.Join(*m, ",")
	return s, nil
}

func (m *Multiaddrs) Scan(src interface{}) error {
	switch src := src.(type) {
	case string:
		s := string(src)
		strs := strings.Split(s, ",")
		*m = append(*m, strs...)
	default:
		return errors.New("invalid multiaddrs")
	}
	return nil
}

type Miner struct {
	ID abi.ActorID `gorm:"primaryKey"`

	// Power 字段可以为空, 因为有些 miner 曾经有算力,但是后来掉光了, 并且,原来的 Power 记录也过期了, 这个时候 power 信息就为空
	Power *PowerInfo `gorm:"-"`
	Peer  *PeerInfo  `gorm:"-"`
	Agent *AgentInfo `gorm:"-"`
}

type PeerInfo struct {
	MinerID    abi.ActorID `gorm:"index"`
	PeerId     string
	Multiaddrs *Multiaddrs
	UpdatedAt  time.Time
}

type PowerInfo struct {
	MinerID         abi.ActorID `gorm:"index"`
	RawBytePower    *Power
	QualityAdjPower *Power
	UpdatedAt       time.Time
}

type AgentInfo struct {
	MinerID   abi.ActorID `gorm:"index"`
	Name      string
	Tag       string `gorm:"index"`
	UpdatedAt time.Time
}

type Api struct {
}

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

func GetBrief(miner Miner) core.MinerBrief {
	qap := 0.0
	if miner.Power != nil {
		qap = util.PiB(miner.Power.QualityAdjPower.String())
	}
	return core.MinerBrief{
		Actor: miner.ID,
		Agent: miner.Agent.Name,
		QAP:   qap,
	}
}
