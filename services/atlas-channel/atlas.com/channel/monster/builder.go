package monster

import (
	"errors"

	"github.com/Chronicle20/atlas-constants/field"
)

var (
	ErrInvalidUniqueId = errors.New("monster unique id must be greater than 0")
)

type modelBuilder struct {
	field              field.Model
	uniqueId           uint32
	maxHp              uint32
	hp                 uint32
	mp                 uint32
	monsterId          uint32
	controlCharacterId uint32
	x                  int16
	y                  int16
	fh                 int16
	stance             byte
	team               int8
}

// NewModelBuilder creates a new builder instance with required fields
func NewModelBuilder(uniqueId uint32, field field.Model, monsterId uint32) *modelBuilder {
	return &modelBuilder{
		field:     field,
		uniqueId:  uniqueId,
		monsterId: monsterId,
	}
}

// CloneModel creates a builder initialized with the Model's values
func CloneModel(m Model) *modelBuilder {
	return &modelBuilder{
		field:              m.field,
		uniqueId:           m.uniqueId,
		maxHp:              m.maxHp,
		hp:                 m.hp,
		mp:                 m.mp,
		monsterId:          m.monsterId,
		controlCharacterId: m.controlCharacterId,
		x:                  m.x,
		y:                  m.y,
		fh:                 m.fh,
		stance:             m.stance,
		team:               m.team,
	}
}

func (b *modelBuilder) SetMaxHP(maxHp uint32) *modelBuilder {
	b.maxHp = maxHp
	return b
}

func (b *modelBuilder) SetHP(hp uint32) *modelBuilder {
	b.hp = hp
	return b
}

func (b *modelBuilder) SetMP(mp uint32) *modelBuilder {
	b.mp = mp
	return b
}

func (b *modelBuilder) SetControlCharacterId(controlCharacterId uint32) *modelBuilder {
	b.controlCharacterId = controlCharacterId
	return b
}

func (b *modelBuilder) SetX(x int16) *modelBuilder {
	b.x = x
	return b
}

func (b *modelBuilder) SetY(y int16) *modelBuilder {
	b.y = y
	return b
}

func (b *modelBuilder) SetStance(stance byte) *modelBuilder {
	b.stance = stance
	return b
}

func (b *modelBuilder) SetFH(fh int16) *modelBuilder {
	b.fh = fh
	return b
}

func (b *modelBuilder) SetTeam(team int8) *modelBuilder {
	b.team = team
	return b
}

// Build creates a new Model instance with validation
func (b *modelBuilder) Build() (Model, error) {
	if b.uniqueId == 0 {
		return Model{}, ErrInvalidUniqueId
	}
	return Model{
		field:              b.field,
		uniqueId:           b.uniqueId,
		maxHp:              b.maxHp,
		hp:                 b.hp,
		mp:                 b.mp,
		monsterId:          b.monsterId,
		controlCharacterId: b.controlCharacterId,
		x:                  b.x,
		y:                  b.y,
		fh:                 b.fh,
		stance:             b.stance,
		team:               b.team,
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
