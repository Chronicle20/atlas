package collection

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-constants/monster"
)

type Builder struct {
	tenantId         uuid.UUID
	characterId      character.Id
	coverCardId      item.Id
	coverMobId       monster.Id
	bookLevel        uint16
	normalCount      uint16
	specialCount     uint16
	expBonusPercent  uint16
	lastCoverEventId *uuid.UUID
	createdAt        time.Time
	updatedAt        time.Time
}

func NewBuilder() *Builder { return &Builder{} }

func CloneBuilder(m Model) *Builder {
	return &Builder{
		tenantId:         m.tenantId,
		characterId:      m.characterId,
		coverCardId:      m.coverCardId,
		coverMobId:       m.coverMobId,
		bookLevel:        m.bookLevel,
		normalCount:      m.normalCount,
		specialCount:     m.specialCount,
		expBonusPercent:  m.expBonusPercent,
		lastCoverEventId: m.lastCoverEventId,
		createdAt:        m.createdAt,
		updatedAt:        m.updatedAt,
	}
}

func (b *Builder) SetTenantId(v uuid.UUID) *Builder       { b.tenantId = v; return b }
func (b *Builder) SetCharacterId(v character.Id) *Builder { b.characterId = v; return b }
func (b *Builder) SetCoverCardId(v item.Id) *Builder      { b.coverCardId = v; return b }
func (b *Builder) SetCoverMobId(v monster.Id) *Builder    { b.coverMobId = v; return b }
func (b *Builder) SetBookLevel(v uint16) *Builder         { b.bookLevel = v; return b }
func (b *Builder) SetNormalCount(v uint16) *Builder       { b.normalCount = v; return b }
func (b *Builder) SetSpecialCount(v uint16) *Builder      { b.specialCount = v; return b }
func (b *Builder) SetExpBonusPercent(v uint16) *Builder   { b.expBonusPercent = v; return b }

func (b *Builder) SetLastCoverEventId(v *uuid.UUID) *Builder {
	b.lastCoverEventId = v
	return b
}
func (b *Builder) SetCreatedAt(v time.Time) *Builder { b.createdAt = v; return b }
func (b *Builder) SetUpdatedAt(v time.Time) *Builder { b.updatedAt = v; return b }

func (b *Builder) Build() (Model, error) {
	if b.characterId == 0 {
		return Model{}, errors.New("characterId is required")
	}
	return Model{
		tenantId:         b.tenantId,
		characterId:      b.characterId,
		coverCardId:      b.coverCardId,
		coverMobId:       b.coverMobId,
		bookLevel:        b.bookLevel,
		normalCount:      b.normalCount,
		specialCount:     b.specialCount,
		expBonusPercent:  b.expBonusPercent,
		lastCoverEventId: b.lastCoverEventId,
		createdAt:        b.createdAt,
		updatedAt:        b.updatedAt,
	}, nil
}

func (b *Builder) MustBuild() Model {
	m, err := b.Build()
	if err != nil {
		panic("MustBuild: " + err.Error())
	}
	return m
}

// Make is the entity → Model adapter used by EntityProvider.
func Make(e entity) (Model, error) {
	return NewBuilder().
		SetTenantId(e.TenantId).
		SetCharacterId(character.Id(e.CharacterId)).
		SetCoverCardId(item.Id(e.CoverCardId)).
		SetCoverMobId(monster.Id(e.CoverMobId)).
		SetBookLevel(e.BookLevel).
		SetNormalCount(e.NormalCount).
		SetSpecialCount(e.SpecialCount).
		SetExpBonusPercent(e.ExpBonusPercent).
		SetLastCoverEventId(e.LastCoverEventId).
		SetCreatedAt(e.CreatedAt).
		SetUpdatedAt(e.UpdatedAt).
		Build()
}
