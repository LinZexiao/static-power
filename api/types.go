package api

import (
	"database/sql/driver"
	"errors"
	"strings"
	"time"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
)

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
	MinerID         abi.ActorID `gorm:"index"`
	RawBytePower    *Power
	QualityAdjPower *Power
	UpdatedAt       time.Time
}

type AgentInfo struct {
	MinerID   abi.ActorID `gorm:"index"`
	Name      string
	UpdatedAt time.Time
}

type Api struct {
}
