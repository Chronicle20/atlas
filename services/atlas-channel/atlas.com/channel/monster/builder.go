package monster

import (
	"errors"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

var ErrInvalidUniqueId = errors.New("monster unique id must be greater than 0")

type builder struct {
	field              field.Model
	uniqueId           uint32
	maxHp              uint32
	hp                 uint32
	mp                 uint32
	maxMp              uint32
	monsterId          uint32
	controlCharacterId uint32
	controllerHasAggro bool
	x                  int16
	y                  int16
	fh                 int16
	stance             byte
	team               int8
	statusEffects      []StatusEffectEntry
}

// NewBuilder creates a new builder instance with required fields
func NewBuilder(uniqueId uint32, field field.Model, monsterId uint32) *builder {
	return &builder{
		field:     field,
		uniqueId:  uniqueId,
		monsterId: monsterId,
	}
}

// CloneModel creates a builder initialized with the Model's values
func CloneModel(m Model) *builder {
	return &builder{
		field:              m.field,
		uniqueId:           m.uniqueId,
		maxHp:              m.maxHp,
		hp:                 m.hp,
		mp:                 m.mp,
		maxMp:              m.maxMp,
		monsterId:          m.monsterId,
		controlCharacterId: m.controlCharacterId,
		controllerHasAggro: m.controllerHasAggro,
		x:                  m.x,
		y:                  m.y,
		fh:                 m.fh,
		stance:             m.stance,
		team:               m.team,
		statusEffects:      m.statusEffects,
	}
}

func (b *builder) SetMaxHp(maxHp uint32) *builder {
	b.maxHp = maxHp
	return b
}

func (b *builder) SetHp(hp uint32) *builder {
	b.hp = hp
	return b
}

func (b *builder) SetMp(mp uint32) *builder {
	b.mp = mp
	return b
}

func (b *builder) SetMaxMp(maxMp uint32) *builder {
	b.maxMp = maxMp
	return b
}

func (b *builder) SetControlCharacterId(controlCharacterId uint32) *builder {
	b.controlCharacterId = controlCharacterId
	return b
}

// SetControllerHasAggro sets whether the controlling character currently has
// aggro on this monster.
func (b *builder) SetControllerHasAggro(aggro bool) *builder {
	b.controllerHasAggro = aggro
	return b
}

func (b *builder) SetX(x int16) *builder {
	b.x = x
	return b
}

func (b *builder) SetY(y int16) *builder {
	b.y = y
	return b
}

func (b *builder) SetStance(stance byte) *builder {
	b.stance = stance
	return b
}

func (b *builder) SetFH(fh int16) *builder {
	b.fh = fh
	return b
}

func (b *builder) SetTeam(team int8) *builder {
	b.team = team
	return b
}

// Build creates a new Model instance with validation
func (b *builder) Build() (Model, error) {
	if b.uniqueId == 0 {
		return Model{}, ErrInvalidUniqueId
	}
	return Model{
		field:              b.field,
		uniqueId:           b.uniqueId,
		maxHp:              b.maxHp,
		hp:                 b.hp,
		mp:                 b.mp,
		maxMp:              b.maxMp,
		monsterId:          b.monsterId,
		controlCharacterId: b.controlCharacterId,
		controllerHasAggro: b.controllerHasAggro,
		x:                  b.x,
		y:                  b.y,
		fh:                 b.fh,
		stance:             b.stance,
		team:               b.team,
		statusEffects:      b.statusEffects,
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
