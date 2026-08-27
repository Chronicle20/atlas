package card

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
)

type Builder struct {
	tenantId        uuid.UUID
	characterId     character.Id
	cardId          item.Id
	level           uint8
	lastEventId     *uuid.UUID
	firstAcquiredAt time.Time
	updatedAt       time.Time
}

func NewBuilder() *Builder { return &Builder{} }

func (b *Builder) SetTenantId(v uuid.UUID) *Builder        { b.tenantId = v; return b }
func (b *Builder) SetCharacterId(v character.Id) *Builder  { b.characterId = v; return b }
func (b *Builder) SetCardId(v item.Id) *Builder            { b.cardId = v; return b }
func (b *Builder) SetLevel(v uint8) *Builder               { b.level = v; return b }
func (b *Builder) SetLastEventId(v *uuid.UUID) *Builder    { b.lastEventId = v; return b }
func (b *Builder) SetFirstAcquiredAt(v time.Time) *Builder { b.firstAcquiredAt = v; return b }
func (b *Builder) SetUpdatedAt(v time.Time) *Builder       { b.updatedAt = v; return b }

func (b *Builder) Build() (Model, error) {
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

func (b *Builder) MustBuild() Model {
	m, err := b.Build()
	if err != nil {
		panic("MustBuild: " + err.Error())
	}
	return m
}

func Make(e entity) (Model, error) {
	return NewBuilder().
		SetTenantId(e.TenantId).
		SetCharacterId(character.Id(e.CharacterId)).
		SetCardId(item.Id(e.CardId)).
		SetLevel(e.Level).
		SetLastEventId(e.LastEventId).
		SetFirstAcquiredAt(e.FirstAcquiredAt).
		SetUpdatedAt(e.UpdatedAt).
		Build()
}
