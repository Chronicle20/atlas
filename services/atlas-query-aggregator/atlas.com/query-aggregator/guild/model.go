package guild

import (
	"atlas-query-aggregator/guild/member"
	"atlas-query-aggregator/guild/title"
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

func (m Model) Logo() uint16 {
	return m.logo
}

func (m Model) LogoColor() byte {
	return m.logoColor
}

func (m Model) LogoBackground() uint16 {
	return m.logoBackground
}

func (m Model) LogoBackgroundColor() byte {
	return m.logoBackgroundColor
}

func (m Model) Titles() []title.Model {
	return m.titles
}

func (m Model) Members() []member.Model {
	return m.members

}

func (m Model) Capacity() uint32 {
	return m.capacity
}

func (m Model) Notice() string {
	return m.notice
}

func (m Model) Points() uint32 {
	return m.points
}

func (m Model) AllianceId() uint32 {
	return 0
}

func (m Model) LeaderId() uint32 {
	return m.leaderId
}

func (m Model) MemberRank(characterId uint32) int {
	for _, mem := range m.Members() {
		if mem.CharacterId() == characterId {
			return int(mem.Rank())
		}
	}
	return 0
}
