package member

import "github.com/google/uuid"

type Model struct {
	tenantId     uuid.UUID
	characterId  uint32
	guildId      uint32
	name         string
	jobId        uint16
	level        byte
	rank         byte
	online       bool
	allianceRank byte
}

func (m Model) CharacterId() uint32 {
	return m.characterId
}
