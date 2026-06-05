package collection

import (
	"errors"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/google/uuid"
)

type ModelBuilder struct {
	tenantId         uuid.UUID
	characterId      character.Id
	coverCardId      item.Id
	coverMobId       uint32
	bookLevel        uint16
	normalCount      uint16
	specialCount     uint16
	expBonusPercent  uint16
	lastCoverEventId *uuid.UUID
	createdAt        time.Time
	updatedAt        time.Time
}

func NewModelBuilder() *ModelBuilder { return &ModelBuilder{} }

func CloneModelBuilder(m Model) *ModelBuilder {
	return &ModelBuilder{
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

func (b *ModelBuilder) SetTenantId(v uuid.UUID) *ModelBuilder         { b.tenantId = v; return b }
func (b *ModelBuilder) SetCharacterId(v character.Id) *ModelBuilder   { b.characterId = v; return b }
func (b *ModelBuilder) SetCoverCardId(v item.Id) *ModelBuilder        { b.coverCardId = v; return b }
func (b *ModelBuilder) SetCoverMobId(v uint32) *ModelBuilder          { b.coverMobId = v; return b }
func (b *ModelBuilder) SetBookLevel(v uint16) *ModelBuilder           { b.bookLevel = v; return b }
func (b *ModelBuilder) SetNormalCount(v uint16) *ModelBuilder         { b.normalCount = v; return b }
func (b *ModelBuilder) SetSpecialCount(v uint16) *ModelBuilder        { b.specialCount = v; return b }
func (b *ModelBuilder) SetExpBonusPercent(v uint16) *ModelBuilder     { b.expBonusPercent = v; return b }
func (b *ModelBuilder) SetLastCoverEventId(v *uuid.UUID) *ModelBuilder { b.lastCoverEventId = v; return b }
func (b *ModelBuilder) SetCreatedAt(v time.Time) *ModelBuilder        { b.createdAt = v; return b }
func (b *ModelBuilder) SetUpdatedAt(v time.Time) *ModelBuilder        { b.updatedAt = v; return b }

func (b *ModelBuilder) Build() (Model, error) {
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

func (b *ModelBuilder) MustBuild() Model {
	m, err := b.Build()
	if err != nil {
		panic("MustBuild: " + err.Error())
	}
	return m
}

// Make is the entity → Model adapter used by EntityProvider.
func Make(e entity) (Model, error) {
	return NewModelBuilder().
		SetTenantId(e.TenantId).
		SetCharacterId(character.Id(e.CharacterId)).
		SetCoverCardId(item.Id(e.CoverCardId)).
		SetCoverMobId(e.CoverMobId).
		SetBookLevel(e.BookLevel).
		SetNormalCount(e.NormalCount).
		SetSpecialCount(e.SpecialCount).
		SetExpBonusPercent(e.ExpBonusPercent).
		SetLastCoverEventId(e.LastCoverEventId).
		SetCreatedAt(e.CreatedAt).
		SetUpdatedAt(e.UpdatedAt).
		Build()
}
