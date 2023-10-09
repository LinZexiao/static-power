package core

import (
	"math"
	"sort"
	"static-power/util"

	"github.com/filecoin-project/go-state-types/abi"
)

func Diff(before, after []MinerBrief) []Difference {
	key := func(m MinerBrief) abi.ActorID {
		return m.Actor
	}
	bf := util.Slice2Map(before, key)
	af := util.Slice2Map(after, key)

	add, keep, rm := util.DiffSet(bf, af)
	diffs := make([]Difference, 0, len(add)+len(keep)+len(rm))

	for actor := range keep {
		qapDiff := af[actor].QAP - bf[actor].QAP
		diffType := QAPChange

		agentBefore := AgentTypeFromString(bf[actor].Agent)
		agentAfter := AgentTypeFromString(af[actor].Agent)
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

func Summarize(miners []MinerBrief) map[AgentType]Summary {
	ret := make(map[AgentType]Summary)
	// divide miners by agent type
	venusMiners := util.SliceFilter(miners, func(miner MinerBrief) bool {
		return AgentTypeFromString(miner.Agent) == AgentTypeVenus
	})
	lotusMiners := util.SliceFilter(miners, func(miner MinerBrief) bool {
		return AgentTypeFromString(miner.Agent) == AgentTypeLotus
	})

	ret[AgentTypeVenus] = summarize(venusMiners)
	ret[AgentTypeLotus] = summarize(lotusMiners)

	return ret
}

func summarize(miners []MinerBrief) Summary {
	ret := Summary{
		Count: (len(miners)),
	}

	for _, miner := range miners {
		ret.QAP += miner.QAP
	}

	return ret
}
