package thread

import (
	"atlas-guilds/thread/reply"
	"github.com/google/uuid"
	"time"
)

type Model struct {
	tenantId   uuid.UUID
	guildId    uint32
	id         uint32
	posterId   uint32
	title      string
	message    string
	emoticonId uint32
	notice     bool
	createdAt  time.Time
	replies    []reply.Model
}

func (m Model) Id() uint32 {
	return m.id
}
