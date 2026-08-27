package skill

import (
	"time"
)

// Builder provides a fluent API for building skill models
type Builder struct {
	id                uint32
	level             byte
	masterLevel       byte
	expiration        time.Time
	cooldownExpiresAt time.Time
}

// NewBuilder creates a new Builder
func NewBuilder() *Builder {
	return &Builder{}
}

// SetId sets the skill ID
func (b *Builder) SetId(id uint32) *Builder {
	b.id = id
	return b
}

// SetLevel sets the skill level
func (b *Builder) SetLevel(level byte) *Builder {
	b.level = level
	return b
}

// SetMasterLevel sets the master level
func (b *Builder) SetMasterLevel(masterLevel byte) *Builder {
	b.masterLevel = masterLevel
	return b
}

// SetExpiration sets the expiration time
func (b *Builder) SetExpiration(expiration time.Time) *Builder {
	b.expiration = expiration
	return b
}

// SetCooldownExpiresAt sets the cooldown expiration time
func (b *Builder) SetCooldownExpiresAt(cooldownExpiresAt time.Time) *Builder {
	b.cooldownExpiresAt = cooldownExpiresAt
	return b
}

// Build creates the Model from the builder
func (b *Builder) Build() Model {
	return Model{
		id:                b.id,
		level:             b.level,
		masterLevel:       b.masterLevel,
		expiration:        b.expiration,
		cooldownExpiresAt: b.cooldownExpiresAt,
	}
}
