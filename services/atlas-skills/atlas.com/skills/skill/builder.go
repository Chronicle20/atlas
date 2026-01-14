package skill

import (
	"errors"
	"time"
)

type modelBuilder struct {
	id                uint32
	level             byte
	masterLevel       byte
	expiration        time.Time
	cooldownExpiresAt time.Time
}

func NewModelBuilder() *modelBuilder {
	return &modelBuilder{}
}

func CloneModel(m Model) *modelBuilder {
	return &modelBuilder{
		id:                m.id,
		level:             m.level,
		masterLevel:       m.masterLevel,
		expiration:        m.expiration,
		cooldownExpiresAt: m.cooldownExpiresAt,
	}
}

func (b *modelBuilder) SetId(id uint32) *modelBuilder {
	b.id = id
	return b
}

func (b *modelBuilder) SetLevel(level byte) *modelBuilder {
	b.level = level
	return b
}

func (b *modelBuilder) SetMasterLevel(masterLevel byte) *modelBuilder {
	b.masterLevel = masterLevel
	return b
}

func (b *modelBuilder) SetExpiration(expiration time.Time) *modelBuilder {
	b.expiration = expiration
	return b
}

func (b *modelBuilder) SetCooldownExpiresAt(cooldownExpiresAt time.Time) *modelBuilder {
	b.cooldownExpiresAt = cooldownExpiresAt
	return b
}

func (b *modelBuilder) Build() (Model, error) {
	if b.id == 0 {
		return Model{}, ErrMissingId
	}
	return Model{
		id:                b.id,
		level:             b.level,
		masterLevel:       b.masterLevel,
		expiration:        b.expiration,
		cooldownExpiresAt: b.cooldownExpiresAt,
	}, nil
}

var (
	ErrMissingId = errors.New("skill id is required")
)
