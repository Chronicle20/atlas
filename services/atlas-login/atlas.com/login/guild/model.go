package guild

import "atlas-login/guild/member"

type Model struct {
	id       uint32
	leaderId uint32
	members  []member.Model
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) LeaderId() uint32 {
	return m.leaderId
}

func (m Model) Members() []member.Model {
	return m.members
}

func (m Model) IsLeader(characterId uint32) bool {
	return m.id != 0 && characterId != 0 && m.leaderId == characterId
}
