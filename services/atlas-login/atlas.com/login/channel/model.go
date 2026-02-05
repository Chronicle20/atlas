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

// Builder is used to construct a Model instance
type Builder struct {
	id              uuid.UUID
	worldId         world.Id
	channelId       channel.Id
	ipAddress       string
	port            int
	currentCapacity uint32
	maxCapacity     uint32
	createdAt       time.Time
}

// NewBuilder creates a new Builder instance
func NewBuilder() *Builder {
	return &Builder{
		createdAt: time.Now(),
	}
}

// SetId sets the id field
func (b *Builder) SetId(id uuid.UUID) *Builder {
	b.id = id
	return b
}

// SetWorldId sets the worldId field
func (b *Builder) SetWorldId(worldId world.Id) *Builder {
	b.worldId = worldId
	return b
}

// SetChannelId sets the channelId field
func (b *Builder) SetChannelId(channelId channel.Id) *Builder {
	b.channelId = channelId
	return b
}

// SetIpAddress sets the ipAddress field
func (b *Builder) SetIpAddress(ipAddress string) *Builder {
	b.ipAddress = ipAddress
	return b
}

// SetPort sets the port field
func (b *Builder) SetPort(port int) *Builder {
	b.port = port
	return b
}

// SetCreatedAt sets the createdAt field
func (b *Builder) SetCreatedAt(createdAt time.Time) *Builder {
	b.createdAt = createdAt
	return b
}

// SetCurrentCapacity sets the currentCapacity field
func (b *Builder) SetCurrentCapacity(currentCapacity uint32) *Builder {
	b.currentCapacity = currentCapacity
	return b
}

// SetMaxCapacity sets the maxCapacity field
func (b *Builder) SetMaxCapacity(maxCapacity uint32) *Builder {
	b.maxCapacity = maxCapacity
	return b
}

// Build creates a new Model instance with the Builder's values
func (b *Builder) Build() Model {
	return Model{
		id:              b.id,
		worldId:         b.worldId,
		channelId:       b.channelId,
		ipAddress:       b.ipAddress,
		port:            b.port,
		currentCapacity: b.currentCapacity,
		maxCapacity:     b.maxCapacity,
		createdAt:       b.createdAt,
	}
}

// ToBuilder creates a Builder initialized with the Model's values
func (m Model) ToBuilder() *Builder {
	return NewBuilder().
		SetId(m.id).
		SetWorldId(m.worldId).
		SetChannelId(m.channelId).
		SetIpAddress(m.ipAddress).
		SetPort(m.port).
		SetCurrentCapacity(m.currentCapacity).
		SetMaxCapacity(m.maxCapacity).
		SetCreatedAt(m.createdAt)
}
