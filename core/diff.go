package core

import (
	"math"
	"sort"
	"static-power/util"

	"github.com/filecoin-project/go-state-types/abi"
)

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

type MinerForDiff struct {
	Actor abi.ActorID
	Agent string
	// RBP in PiB
	QAP float64
}

// compute diff
type Difference struct {
	Actor    abi.ActorID
	DiffType DiffType
	Agent    AgentType

	// QAP in PiB
	QAP float64
}

func Diff(before, after map[abi.ActorID]MinerForDiff) []Difference {
	add, keep, rm := util.DiffSet(before, after)
	diffs := make([]Difference, 0, len(add)+len(keep)+len(rm))

	for actor := range keep {
		qapDiff := after[actor].QAP - before[actor].QAP
		diffType := QAPChange

		agentBefore := AgentTypeFromString(before[actor].Agent)
		agentAfter := AgentTypeFromString(after[actor].Agent)
		if agentAfter != agentBefore {
			diffType = AgentChanged
		}

		if math.Abs(qapDiff) < 0.0000000001 && diffType == QAPChange {
			continue
		}

		diffs = append(diffs, Difference{
			Actor:    actor,
			QAP:      qapDiff,
			DiffType: diffType,
			Agent:    agentAfter,
		})
	}
	for actor, miner := range add {
		diffs = append(diffs, Difference{
			Actor:    actor,
			QAP:      miner.QAP,
			DiffType: Added,
			Agent:    AgentTypeFromString(miner.Agent),
		})
	}
	for actor, miner := range rm {
		diffs = append(diffs, Difference{
			Actor:    actor,
			QAP:      -miner.QAP,
			DiffType: Removed,
			Agent:    AgentTypeFromString(miner.Agent),
		})
	}

	sort.Slice(diffs, func(i, j int) bool {
		if diffs[i].Agent != diffs[j].Agent {
			return diffs[i].Agent > diffs[j].Agent
		} else if diffs[i].DiffType != diffs[j].DiffType {
			return diffs[i].DiffType > diffs[j].DiffType
		} else if math.Abs(diffs[i].QAP) != math.Abs(diffs[j].QAP) {
			return math.Abs(diffs[i].QAP) > math.Abs(diffs[j].QAP)
		}
		return diffs[i].Actor > diffs[j].Actor
	})

	return diffs
}
