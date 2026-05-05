package card

import (
	"errors"
	"fmt"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/google/uuid"
)

type ModelBuilder struct {
	tenantId        uuid.UUID
	characterId     character.Id
	cardId          item.Id
	level           uint8
	lastEventId     *uuid.UUID
	firstAcquiredAt time.Time
	updatedAt       time.Time
}

func NewModelBuilder() *ModelBuilder { return &ModelBuilder{} }

func (b *ModelBuilder) SetTenantId(v uuid.UUID) *ModelBuilder        { b.tenantId = v; return b }
func (b *ModelBuilder) SetCharacterId(v character.Id) *ModelBuilder  { b.characterId = v; return b }
func (b *ModelBuilder) SetCardId(v item.Id) *ModelBuilder            { b.cardId = v; return b }
func (b *ModelBuilder) SetLevel(v uint8) *ModelBuilder               { b.level = v; return b }
func (b *ModelBuilder) SetLastEventId(v *uuid.UUID) *ModelBuilder    { b.lastEventId = v; return b }
func (b *ModelBuilder) SetFirstAcquiredAt(v time.Time) *ModelBuilder { b.firstAcquiredAt = v; return b }
func (b *ModelBuilder) SetUpdatedAt(v time.Time) *ModelBuilder       { b.updatedAt = v; return b }

func (b *ModelBuilder) Build() (Model, error) {
	if b.characterId == 0 {
		return Model{}, errors.New("characterId is required")
	}
	if !IsCardId(b.cardId) {
		return Model{}, fmt.Errorf("cardId %d is not a monster-book card item", b.cardId)
	}
	if b.level < 1 || b.level > MaxLevel {
		return Model{}, fmt.Errorf("level %d out of range [1, %d]", b.level, MaxLevel)
	}
	return Model{
		tenantId:        b.tenantId,
		characterId:     b.characterId,
		cardId:          b.cardId,
		level:           b.level,
		isSpecial:       IsSpecialCard(b.cardId),
		lastEventId:     b.lastEventId,
		firstAcquiredAt: b.firstAcquiredAt,
		updatedAt:       b.updatedAt,
	}, nil
}

func (b *ModelBuilder) MustBuild() Model {
	m, err := b.Build()
	if err != nil {
		panic("MustBuild: " + err.Error())
	}
	return m
}

func Make(e entity) (Model, error) {
	return NewModelBuilder().
		SetTenantId(e.TenantId).
		SetCharacterId(character.Id(e.CharacterId)).
		SetCardId(item.Id(e.CardId)).
		SetLevel(e.Level).
		SetLastEventId(e.LastEventId).
		SetFirstAcquiredAt(e.FirstAcquiredAt).
		SetUpdatedAt(e.UpdatedAt).
		Build()
}
