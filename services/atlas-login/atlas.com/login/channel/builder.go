package channel

import (
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

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
