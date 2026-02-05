package channel

import (
	"time"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

type Model struct {
	id              uuid.UUID
	worldId         world.Id
	channelId       channel.Id
	ipAddress       string
	port            int
	currentCapacity uint32
	maxCapacity     uint32
	createdAt       time.Time
	expRate         float64
	mesoRate        float64
	itemDropRate    float64
	questExpRate    float64
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

func (m Model) CurrentCapacity() uint32 {
	return m.currentCapacity
}

func (m Model) MaxCapacity() uint32 {
	return m.maxCapacity
}

func (m Model) ExpRate() float64 {
	if m.expRate == 0 {
		return 1.0
	}
	return m.expRate
}

func (m Model) MesoRate() float64 {
	if m.mesoRate == 0 {
		return 1.0
	}
	return m.mesoRate
}

func (m Model) ItemDropRate() float64 {
	if m.itemDropRate == 0 {
		return 1.0
	}
	return m.itemDropRate
}

func (m Model) QuestExpRate() float64 {
	if m.questExpRate == 0 {
		return 1.0
	}
	return m.questExpRate
}
