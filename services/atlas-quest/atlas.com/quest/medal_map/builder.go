package medal_map

import (
	"errors"

	"github.com/google/uuid"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

type builder struct {
	id          uuid.UUID
	characterId uint32
	questId     uint32
	mapId       _map.Id
}

func NewBuilder() *builder {
	return &builder{}
}

func (b *builder) SetId(id uuid.UUID) *builder {
	b.id = id
	return b
}

func (b *builder) SetCharacterId(characterId uint32) *builder {
	b.characterId = characterId
	return b
}

func (b *builder) SetQuestId(questId uint32) *builder {
	b.questId = questId
	return b
}

func (b *builder) SetMapId(mapId _map.Id) *builder {
	b.mapId = mapId
	return b
}

func (b *builder) Build() Model {
	return Model{
		id:          b.id,
		characterId: b.characterId,
		questId:     b.questId,
		mapId:       b.mapId,
	}
}

// BuildWithValidation returns the built Model with validation, returning an
// error if required fields are missing. This is the recommended method for
// new code (matches quest.builder's own remediation for this rule).
func (b *builder) BuildWithValidation() (Model, error) {
	if b.characterId == 0 {
		return Model{}, ErrMissingCharacterId
	}
	if b.questId == 0 {
		return Model{}, ErrMissingQuestId
	}
	return b.Build(), nil
}

// Validation errors for builder
var (
	ErrMissingCharacterId = errors.New("character ID is required")
	ErrMissingQuestId     = errors.New("quest ID is required")
)
