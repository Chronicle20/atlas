package channel

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrMissingId         = errors.New("channel id is required")
	ErrInvalidIpAddress  = errors.New("ip address is required")
	ErrInvalidPort       = errors.New("port must be between 1 and 65535")
	ErrInvalidCapacity   = errors.New("max capacity must be greater than 0")
)

type modelBuilder struct {
	id              uuid.UUID
	worldId         byte
	channelId       byte
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

// NewModelBuilder creates a new builder instance
func NewModelBuilder() *modelBuilder {
	return &modelBuilder{
		createdAt:    time.Now(),
		expRate:      1.0,
		mesoRate:     1.0,
		itemDropRate: 1.0,
		questExpRate: 1.0,
	}
}

// CloneModel creates a builder initialized with the Model's values
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
		expRate:         m.expRate,
		mesoRate:        m.mesoRate,
		itemDropRate:    m.itemDropRate,
		questExpRate:    m.questExpRate,
	}
}

// SetId sets the id field
func (b *modelBuilder) SetId(id uuid.UUID) *modelBuilder {
	b.id = id
	return b
}

// SetWorldId sets the worldId field
func (b *modelBuilder) SetWorldId(worldId byte) *modelBuilder {
	b.worldId = worldId
	return b
}

// SetChannelId sets the channelId field
func (b *modelBuilder) SetChannelId(channelId byte) *modelBuilder {
	b.channelId = channelId
	return b
}

// SetIpAddress sets the ipAddress field
func (b *modelBuilder) SetIpAddress(ipAddress string) *modelBuilder {
	b.ipAddress = ipAddress
	return b
}

// SetPort sets the port field
func (b *modelBuilder) SetPort(port int) *modelBuilder {
	b.port = port
	return b
}

// SetCreatedAt sets the createdAt field
func (b *modelBuilder) SetCreatedAt(createdAt time.Time) *modelBuilder {
	b.createdAt = createdAt
	return b
}

// SetCurrentCapacity sets the currentCapacity field
func (b *modelBuilder) SetCurrentCapacity(currentCapacity uint32) *modelBuilder {
	b.currentCapacity = currentCapacity
	return b
}

// SetMaxCapacity sets the maxCapacity field
func (b *modelBuilder) SetMaxCapacity(maxCapacity uint32) *modelBuilder {
	b.maxCapacity = maxCapacity
	return b
}

// SetExpRate sets the experience rate multiplier
func (b *modelBuilder) SetExpRate(expRate float64) *modelBuilder {
	b.expRate = expRate
	return b
}

// SetMesoRate sets the meso rate multiplier
func (b *modelBuilder) SetMesoRate(mesoRate float64) *modelBuilder {
	b.mesoRate = mesoRate
	return b
}

// SetItemDropRate sets the item drop rate multiplier
func (b *modelBuilder) SetItemDropRate(itemDropRate float64) *modelBuilder {
	b.itemDropRate = itemDropRate
	return b
}

// SetQuestExpRate sets the quest experience rate multiplier
func (b *modelBuilder) SetQuestExpRate(questExpRate float64) *modelBuilder {
	b.questExpRate = questExpRate
	return b
}

// Build creates a new Model instance with validation
func (b *modelBuilder) Build() (Model, error) {
	if b.id == uuid.Nil {
		return Model{}, ErrMissingId
	}
	if b.ipAddress == "" {
		return Model{}, ErrInvalidIpAddress
	}
	if b.port < 1 || b.port > 65535 {
		return Model{}, ErrInvalidPort
	}
	if b.maxCapacity == 0 {
		return Model{}, ErrInvalidCapacity
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
		expRate:         b.expRate,
		mesoRate:        b.mesoRate,
		itemDropRate:    b.itemDropRate,
		questExpRate:    b.questExpRate,
	}, nil
}

// MustBuild creates a new Model instance, panicking on validation error
func (b *modelBuilder) MustBuild() Model {
	m, err := b.Build()
	if err != nil {
		panic(err)
	}
	return m
}
