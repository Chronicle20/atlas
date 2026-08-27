package channel

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

var ErrInvalidId = errors.New("channel id must not be nil")

type builder struct {
	id              uuid.UUID
	worldId         world.Id
	channelId       channel.Id
	ipAddress       string
	port            int
	currentCapacity uint32
	maxCapacity     uint32
	createdAt       time.Time
}

// NewBuilder creates a new builder instance
func NewBuilder() *builder {
	return &builder{
		createdAt: time.Now(),
	}
}

func CloneModel(m Model) *builder {
	return &builder{
		id:              m.id,
		worldId:         m.worldId,
		channelId:       m.channelId,
		ipAddress:       m.ipAddress,
		port:            m.port,
		currentCapacity: m.currentCapacity,
		maxCapacity:     m.maxCapacity,
		createdAt:       m.createdAt,
	}
}

func (b *builder) SetId(id uuid.UUID) *builder {
	b.id = id
	return b
}

func (b *builder) SetWorldId(worldId world.Id) *builder {
	b.worldId = worldId
	return b
}

func (b *builder) SetChannelId(channelId channel.Id) *builder {
	b.channelId = channelId
	return b
}

func (b *builder) SetIpAddress(ipAddress string) *builder {
	b.ipAddress = ipAddress
	return b
}

func (b *builder) SetPort(port int) *builder {
	b.port = port
	return b
}

func (b *builder) SetCreatedAt(createdAt time.Time) *builder {
	b.createdAt = createdAt
	return b
}

func (b *builder) SetCurrentCapacity(currentCapacity uint32) *builder {
	b.currentCapacity = currentCapacity
	return b
}

func (b *builder) SetMaxCapacity(maxCapacity uint32) *builder {
	b.maxCapacity = maxCapacity
	return b
}

func (b *builder) Build() (Model, error) {
	if b.id == uuid.Nil {
		return Model{}, ErrInvalidId
	}
	return Model{
		id:              b.id,
		worldId:         b.worldId,
		channelId:       b.channelId,
		ipAddress:       b.ipAddress,
		port:            b.port,
		currentCapacity: b.currentCapacity,
		maxCapacity:     b.maxCapacity,
		createdAt:       b.createdAt,
	}, nil
}

func (b *builder) MustBuild() Model {
	m, err := b.Build()
	if err != nil {
		panic(err)
	}
	return m
}
