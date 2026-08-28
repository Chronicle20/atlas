package mount

import (
	"time"

	"github.com/google/uuid"
)

// Builder constructs an immutable mount Model. Defaults: level 1, exp 0,
// tiredness 0, nil lastTirednessTickAt. Use Clone to seed a builder from an
// existing Model.
type Builder struct {
	tenantId            uuid.UUID
	characterId         uint32
	id                  uuid.UUID
	level               int
	exp                 int
	tiredness           int
	lastTirednessTickAt *time.Time
}

func NewBuilder(tenantId uuid.UUID, characterId uint32, id uuid.UUID) *Builder {
	return &Builder{
		tenantId:    tenantId,
		characterId: characterId,
		id:          id,
		level:       1,
		exp:         0,
		tiredness:   0,
	}
}

func Clone(m Model) *Builder {
	return NewBuilder(m.TenantId(), m.CharacterId(), m.Id()).
		SetLevel(m.Level()).
		SetExp(m.Exp()).
		SetTiredness(m.Tiredness()).
		SetLastTirednessTickAt(m.LastTirednessTickAt())
}

func (b *Builder) SetLevel(level int) *Builder {
	b.level = level
	return b
}

func (b *Builder) SetExp(exp int) *Builder {
	b.exp = exp
	return b
}

func (b *Builder) SetTiredness(tiredness int) *Builder {
	b.tiredness = tiredness
	return b
}

func (b *Builder) SetLastTirednessTickAt(at *time.Time) *Builder {
	b.lastTirednessTickAt = at
	return b
}

func (b *Builder) Build() (Model, error) {
	return Model{
		tenantId:            b.tenantId,
		characterId:         b.characterId,
		id:                  b.id,
		level:               b.level,
		exp:                 b.exp,
		tiredness:           b.tiredness,
		lastTirednessTickAt: b.lastTirednessTickAt,
	}, nil
}
