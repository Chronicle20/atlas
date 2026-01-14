package channel

import (
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
	"time"
)

type Model struct {
	id              uuid.UUID
	worldId         world.Id
	channelId       channel.Id
	ipAddress       string
	port            int
	currentCapacity int
	maxCapacity     int
	createdAt       time.Time
}

func (m Model) Id() uuid.UUID {
	return m.id
}

func (m Model) WorldId() world.Id {
	return m.worldId
}

func (m Model) ChannelId() channel.Id {
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

func (m Model) CurrentCapacity() int {
	return m.currentCapacity
}

func (m Model) MaxCapacity() int {
	return m.maxCapacity
}
