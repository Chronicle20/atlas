package channel

import (
	"github.com/google/uuid"
	"time"
)

type Model struct {
	id        uuid.UUID
	worldId   byte
	channelId byte
	ipAddress string
	port      int
	createdAt time.Time
}

func (m Model) Id() uuid.UUID {
	return m.id
}

func (m Model) WorldId() byte {
	return m.worldId
}

func (m Model) ChannelId() byte {
	return m.channelId
}

func (m Model) IpAddress() string {
	return m.ipAddress
}

func (m Model) Port() int {
	return m.port
}

func (m Model) CreatedAt() time.Time {
	return m.createdAt
}
