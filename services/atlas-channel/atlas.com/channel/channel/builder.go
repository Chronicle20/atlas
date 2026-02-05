package channel

import (
	"errors"
	"time"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

var (
	ErrInvalidId = errors.New("channel id must not be nil")
)

type modelBuilder struct {
	id              uuid.UUID
	worldId         world.Id
	channelId       channel.Id
	ipAddress       string
	port            int
	currentCapacity uint32
	maxCapacity     uint32
	createdAt       time.Time
}

func NewModelBuilder() *modelBuilder {
	return &modelBuilder{
		createdAt: time.Now(),
	}
}

// NewBuilder is an alias for NewModelBuilder for backward compatibility
func NewBuilder() *modelBuilder {
	return NewModelBuilder()
}

func CloneModel(m Model) *modelBuilder {
	return &modelBuilder{
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

func (b *modelBuilder) SetId(id uuid.UUID) *modelBuilder {
	b.id = id
	return b
}

func (b *modelBuilder) SetWorldId(worldId world.Id) *modelBuilder {
	b.worldId = worldId
	return b
}

func (b *modelBuilder) SetChannelId(channelId channel.Id) *modelBuilder {
	b.channelId = channelId
	return b
}

func (b *modelBuilder) SetIpAddress(ipAddress string) *modelBuilder {
	b.ipAddress = ipAddress
	return b
}

func (b *modelBuilder) SetPort(port int) *modelBuilder {
	b.port = port
	return b
}

func (b *modelBuilder) SetCreatedAt(createdAt time.Time) *modelBuilder {
	b.createdAt = createdAt
	return b
}

func (b *modelBuilder) SetCurrentCapacity(currentCapacity uint32) *modelBuilder {
	b.currentCapacity = currentCapacity
	return b
}

func (b *modelBuilder) SetMaxCapacity(maxCapacity uint32) *modelBuilder {
	b.maxCapacity = maxCapacity
	return b
}

func (b *modelBuilder) Build() (Model, error) {
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

func (b *modelBuilder) MustBuild() Model {
	m, err := b.Build()
	if err != nil {
		panic(err)
	}
	return m
}
