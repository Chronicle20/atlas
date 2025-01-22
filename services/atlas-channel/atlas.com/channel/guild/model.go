package guild

import (
	"atlas-channel/guild/member"
	"atlas-channel/guild/title"
)

type Model struct {
	id                  uint32
	worldId             byte
	name                string
	notice              string
	points              uint32
	capacity            uint32
	logo                uint16
	logoColor           byte
	logoBackground      uint16
	logoBackgroundColor byte
	leaderId            uint32
	members             []member.Model
	titles              []title.Model
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) Name() string {
	return m.name
}
