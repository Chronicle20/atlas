package title

import "github.com/google/uuid"

type Model struct {
	tenantId uuid.UUID
	id       uuid.UUID
	guildId  uint32
	name     string
	index    byte
}
