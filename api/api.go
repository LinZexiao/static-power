package api

import (
	"database/sql/driver"
	"errors"
	"strings"
	"time"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"gorm.io/gorm"
)

var NetWork abi.ActorID = 0

var db *gorm.DB = nil

type Power big.Int

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
	ID    abi.ActorID `gorm:"primaryKey"`
	Power *PowerInfo  `gorm:"-"`
	Peer  *PeerInfo   `gorm:"-"`
	Agent *AgentInfo  `gorm:"-"`
}

type PeerInfo struct {
	MinerID    abi.ActorID `gorm:"index"`
	PeerId     string
	Multiaddrs *Multiaddrs
	UpdatedAt  time.Time
}

type PowerInfo struct {
	MinerID                  abi.ActorID `gorm:"index"`
	RawBytePower             *Power
	QualityAdjustedBytePower *Power
	UpdatedAt                time.Time
}

type AgentInfo struct {
	MinerID   abi.ActorID `gorm:"index"`
	Name      string
	UpdatedAt time.Time
}

type Api struct {
}

func NewApi(d *gorm.DB) *Api {
	d.AutoMigrate(&Miner{}, &PeerInfo{}, &PowerInfo{}, &AgentInfo{})
	db = d
	return &Api{}
}

func (a *Api) GetProportion() (float32, error) {
	return 0.0, nil
}

// get miner info to query agent
func (a *Api) GetMinerInfo() ([]Miner, error) {
	var miners []Miner
	db.Find(&miners)
	for i := range miners {
		miner := &miners[i]
		var peer PeerInfo
		err := db.Order("updated_at desc").First(&peer, "miner_id = ?", miner.ID).Error
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
	}
	return miners, nil
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
