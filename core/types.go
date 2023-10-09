package core

import (
	"strings"

	"github.com/filecoin-project/go-state-types/abi"
)

const (
	MinerCsvVersion = "1.0"
	DiffCsvVersion  = "2.0"
)

func CsvVersion2Skip(version string) int {
	switch version {
	case MinerCsvVersion:
		return 4
	case DiffCsvVersion:
		return 4
	default:
		return 0
	}
}

type AgentType uint8

const (
	AgentTypeOther AgentType = iota
	AgentTypeLotus
	AgentTypeVenus
)

func (a AgentType) String() string {
	switch a {
	case AgentTypeLotus:
		return "lotus"
	case AgentTypeVenus:
		return "venus"
	default:
		return "others"
	}
}

func AgentTypeFromString(agent string) AgentType {
	if strings.Contains(agent, "droplet") || strings.Contains(agent, "venus") || strings.Contains(agent, "market") {
		return AgentTypeVenus
	} else if strings.Contains(agent, "lotus") || strings.Contains(agent, "boost") {
		return AgentTypeLotus
	} else {
		return AgentTypeOther
	}
}

type DiffType uint8

const (
	UnKnown DiffType = iota
	QAPChange
	Added
	Removed

	// indicates that the agent change between before and after
	AgentChanged
)

func (d DiffType) String() string {
	switch d {
	case Added:
		return "added"
	case Removed:
		return "removed"
	case QAPChange:
		return "qap_changed"
	case AgentChanged:
		return "agent_changed"
	default:
		return "unknown"
	}
}

type MinerBrief struct {
	Actor abi.ActorID
	Agent string
	// QAP in PiB
	QAP float64
}

type Difference struct {
	Actor    abi.ActorID
	DiffType DiffType
	Agent    AgentType

	// QAP in PiB
	QAP float64
}

type Summary struct {
	Count int
	QAP   float64
}
