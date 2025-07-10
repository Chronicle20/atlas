package family

import (
	"time"

	"github.com/google/uuid"
)

// NewBuilder creates a new builder with required parameters
func NewBuilder(characterId uint32, tenantId uuid.UUID, level uint16, world byte, channel byte, mapId uint32) *Builder {
	return &Builder{
		characterId: characterId,
		tenantId:    tenantId,
		level:       level,
		world:       world,
		channel:     channel,
		mapId:       mapId,
		juniorIds:   []uint32{},
		createdAt:   time.Now(),
		updatedAt:   time.Now(),
	}
}

// Fluent setters for optional parameters

func (b *Builder) SetId(id uint32) *Builder {
	b.id = id
	return b
}

func (b *Builder) SetSeniorId(seniorId uint32) *Builder {
	b.seniorId = &seniorId
	return b
}

func (b *Builder) ClearSeniorId() *Builder {
	b.seniorId = nil
	return b
}

func (b *Builder) SetJuniorIds(juniorIds []uint32) *Builder {
	// Copy to avoid shared references
	b.juniorIds = make([]uint32, len(juniorIds))
	copy(b.juniorIds, juniorIds)
	return b
}

func (b *Builder) AddJunior(juniorId uint32) *Builder {
	b.juniorIds = append(b.juniorIds, juniorId)
	return b
}

func (b *Builder) RemoveJunior(juniorId uint32) *Builder {
	for i, id := range b.juniorIds {
		if id == juniorId {
			b.juniorIds = append(b.juniorIds[:i], b.juniorIds[i+1:]...)
			break
		}
	}
	return b
}

func (b *Builder) SetRep(rep uint32) *Builder {
	b.rep = rep
	return b
}

func (b *Builder) AddRep(amount uint32) *Builder {
	b.rep += amount
	return b
}

func (b *Builder) SubtractRep(amount uint32) *Builder {
	if b.rep >= amount {
		b.rep -= amount
	} else {
		b.rep = 0
	}
	return b
}

func (b *Builder) SetDailyRep(dailyRep uint32) *Builder {
	b.dailyRep = dailyRep
	return b
}

func (b *Builder) AddDailyRep(amount uint32) *Builder {
	b.dailyRep += amount
	return b
}

func (b *Builder) ResetDailyRep() *Builder {
	b.dailyRep = 0
	return b
}

func (b *Builder) SetLevel(level uint16) *Builder {
	b.level = level
	return b
}

func (b *Builder) SetWorld(world byte) *Builder {
	b.world = world
	return b
}

func (b *Builder) SetChannel(channel byte) *Builder {
	b.channel = channel
	return b
}

func (b *Builder) SetMapId(mapId uint32) *Builder {
	b.mapId = mapId
	return b
}

func (b *Builder) SetCreatedAt(createdAt time.Time) *Builder {
	b.createdAt = createdAt
	return b
}

func (b *Builder) SetUpdatedAt(updatedAt time.Time) *Builder {
	b.updatedAt = updatedAt
	return b
}

func (b *Builder) Touch() *Builder {
	b.updatedAt = time.Now()
	return b
}

// Build validates business rules and constructs the final immutable FamilyMember
func (b *Builder) Build() (FamilyMember, error) {
	// Validate required fields
	if err := ValidateCharacterId(b.characterId); err != nil {
		return FamilyMember{}, err
	}

	if err := ValidateTenantId(b.tenantId); err != nil {
		return FamilyMember{}, err
	}

	if err := ValidateLevel(b.level); err != nil {
		return FamilyMember{}, err
	}

	if err := ValidateJuniorIds(b.characterId, b.juniorIds); err != nil {
		return FamilyMember{}, err
	}

	if err := ValidateSeniorId(b.characterId, b.seniorId); err != nil {
		return FamilyMember{}, err
	}

	// Business rule validations
	if b.dailyRep > 5000 {
		return FamilyMember{}, ErrInvalidDailyRep
	}

	// Copy junior IDs to avoid shared references
	juniorIds := make([]uint32, len(b.juniorIds))
	copy(juniorIds, b.juniorIds)

	return FamilyMember{
		id:          b.id,
		characterId: b.characterId,
		tenantId:    b.tenantId,
		seniorId:    b.seniorId,
		juniorIds:   juniorIds,
		rep:         b.rep,
		dailyRep:    b.dailyRep,
		level:       b.level,
		world:       b.world,
		channel:     b.channel,
		mapId:       b.mapId,
		createdAt:   b.createdAt,
		updatedAt:   b.updatedAt,
	}, nil
}