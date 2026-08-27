package skill

import (
	"errors"
	"time"
)

type builder struct {
	id                uint32
	level             byte
	masterLevel       byte
	expiration        time.Time
	cooldownExpiresAt time.Time
}

func NewBuilder() *builder {
	return &builder{}
}

func CloneModel(m Model) *builder {
	return &builder{
		id:                m.id,
		level:             m.level,
		masterLevel:       m.masterLevel,
		expiration:        m.expiration,
		cooldownExpiresAt: m.cooldownExpiresAt,
	}
}

func (b *builder) SetId(id uint32) *builder {
	b.id = id
	return b
}

func (b *builder) SetLevel(level byte) *builder {
	b.level = level
	return b
}

func (b *builder) SetMasterLevel(masterLevel byte) *builder {
	b.masterLevel = masterLevel
	return b
}

func (b *builder) SetExpiration(expiration time.Time) *builder {
	b.expiration = expiration
	return b
}

func (b *builder) SetCooldownExpiresAt(cooldownExpiresAt time.Time) *builder {
	b.cooldownExpiresAt = cooldownExpiresAt
	return b
}

func (b *builder) Build() (Model, error) {
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

var ErrMissingId = errors.New("skill id is required")
