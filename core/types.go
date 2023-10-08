package core

import "strings"

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
