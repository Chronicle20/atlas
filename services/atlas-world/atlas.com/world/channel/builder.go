package channel

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

var (
	ErrMissingId        = errors.New("channel id is required")
	ErrInvalidIpAddress = errors.New("ip address is required")
	ErrInvalidPort      = errors.New("port must be between 1 and 65535")
	ErrInvalidCapacity  = errors.New("max capacity must be greater than 0")
)

type builder struct {
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

// NewBuilder creates a new builder instance
func NewBuilder() *builder {
	return &builder{
		createdAt:    time.Now(),
		expRate:      1.0,
		mesoRate:     1.0,
		itemDropRate: 1.0,
		questExpRate: 1.0,
	}
}

// CloneModel creates a builder initialized with the Model's values
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
		expRate:         m.expRate,
		mesoRate:        m.mesoRate,
		itemDropRate:    m.itemDropRate,
		questExpRate:    m.questExpRate,
	}
}

// SetId sets the id field
func (b *builder) SetId(id uuid.UUID) *builder {
	b.id = id
	return b
}

// SetWorldId sets the worldId field
func (b *builder) SetWorldId(worldId world.Id) *builder {
	b.worldId = worldId
	return b
}

// SetChannelId sets the channelId field
func (b *builder) SetChannelId(channelId channel.Id) *builder {
	b.channelId = channelId
	return b
}

// SetIpAddress sets the ipAddress field
func (b *builder) SetIpAddress(ipAddress string) *builder {
	b.ipAddress = ipAddress
	return b
}

// SetPort sets the port field
func (b *builder) SetPort(port int) *builder {
	b.port = port
	return b
}

// SetCreatedAt sets the createdAt field
func (b *builder) SetCreatedAt(createdAt time.Time) *builder {
	b.createdAt = createdAt
	return b
}

// SetCurrentCapacity sets the currentCapacity field
func (b *builder) SetCurrentCapacity(currentCapacity uint32) *builder {
	b.currentCapacity = currentCapacity
	return b
}

// SetMaxCapacity sets the maxCapacity field
func (b *builder) SetMaxCapacity(maxCapacity uint32) *builder {
	b.maxCapacity = maxCapacity
	return b
}

// SetExpRate sets the experience rate multiplier
func (b *builder) SetExpRate(expRate float64) *builder {
	b.expRate = expRate
	return b
}

// SetMesoRate sets the meso rate multiplier
func (b *builder) SetMesoRate(mesoRate float64) *builder {
	b.mesoRate = mesoRate
	return b
}

// SetItemDropRate sets the item drop rate multiplier
func (b *builder) SetItemDropRate(itemDropRate float64) *builder {
	b.itemDropRate = itemDropRate
	return b
}

// SetQuestExpRate sets the quest experience rate multiplier
func (b *builder) SetQuestExpRate(questExpRate float64) *builder {
	b.questExpRate = questExpRate
	return b
}

// Build creates a new Model instance with validation
func (b *builder) Build() (Model, error) {
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
func (b *builder) MustBuild() Model {
	m, err := b.Build()
	if err != nil {
		panic(err)
	}
	return m
}
