package skill

import (
	"errors"
	"time"

	skillconst "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

var ErrInvalidSkillId = errors.New("skill id must be greater than 0")

type modelBuilder struct {
	id                skillconst.Id
	level             byte
	masterLevel       byte
	expiration        time.Time
	cooldownExpiresAt time.Time
}

// NewModelBuilder creates a new builder instance
func NewModelBuilder(id skillconst.Id) *modelBuilder {
	return &modelBuilder{id: id}
}

// Clone creates a builder initialized with the Model's values
func Clone(m Model) *modelBuilder {
	return &modelBuilder{
		id:                m.id,
		level:             m.level,
		masterLevel:       m.masterLevel,
		expiration:        m.expiration,
		cooldownExpiresAt: m.cooldownExpiresAt,
	}
}

func (b *modelBuilder) SetLevel(v byte) *modelBuilder       { b.level = v; return b }
func (b *modelBuilder) SetMasterLevel(v byte) *modelBuilder { b.masterLevel = v; return b }
func (b *modelBuilder) SetExpiration(v time.Time) *modelBuilder {
	b.expiration = v
	return b
}

func (b *modelBuilder) SetCooldownExpiresAt(v time.Time) *modelBuilder {
	b.cooldownExpiresAt = v
	return b
}

func (b *modelBuilder) Build() (Model, error) {
	if b.id == 0 {
		return Model{}, ErrInvalidSkillId
	}
	return Model{
		id:                b.id,
		level:             b.level,
		masterLevel:       b.masterLevel,
		expiration:        b.expiration,
		cooldownExpiresAt: b.cooldownExpiresAt,
	}, nil
}

func (b *modelBuilder) MustBuild() Model {
	m, err := b.Build()
	if err != nil {
		panic(err)
	}
	return m
}
