package character

import "github.com/google/uuid"

type Model struct {
	tenantId    uuid.UUID
	characterId uint32
	guildId     uint32
}
