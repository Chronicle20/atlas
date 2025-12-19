package character

import "github.com/google/uuid"

type Model struct {
	tenantId    uuid.UUID
	characterId uint32
	guildId     uint32
}

func (m Model) GuildId() uint32 {
	return m.guildId
}
