package mount

import (
	"time"

	"github.com/google/uuid"
)

// ModelBuilder constructs an immutable mount Model. Defaults: level 1, exp 0,
// tiredness 0, nil lastTirednessTickAt. Use Clone to seed a builder from an
// existing Model.
type ModelBuilder struct {
	tenantId            uuid.UUID
	characterId         uint32
	id                  uuid.UUID
	level               int
	exp                 int
	tiredness           int
	lastTirednessTickAt *time.Time
}

func NewModelBuilder(tenantId uuid.UUID, characterId uint32, id uuid.UUID) *ModelBuilder {
	return &ModelBuilder{
		tenantId:    tenantId,
		characterId: characterId,
		id:          id,
		level:       1,
		exp:         0,
		tiredness:   0,
	}
}

func Clone(m Model) *ModelBuilder {
	return NewModelBuilder(m.TenantId(), m.CharacterId(), m.Id()).
		SetLevel(m.Level()).
		SetExp(m.Exp()).
		SetTiredness(m.Tiredness()).
		SetLastTirednessTickAt(m.LastTirednessTickAt())
}

func (b *ModelBuilder) SetLevel(level int) *ModelBuilder {
	b.level = level
	return b
}

func (b *ModelBuilder) SetExp(exp int) *ModelBuilder {
	b.exp = exp
	return b
}

func (b *ModelBuilder) SetTiredness(tiredness int) *ModelBuilder {
	b.tiredness = tiredness
	return b
}

func (b *ModelBuilder) SetLastTirednessTickAt(at *time.Time) *ModelBuilder {
	b.lastTirednessTickAt = at
	return b
}

func (b *ModelBuilder) Build() (Model, error) {
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
