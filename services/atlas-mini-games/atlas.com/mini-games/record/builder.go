package record

import (
	"errors"

	"github.com/google/uuid"
)

// Builder constructs an immutable Model. Used for zero-filled/absent-row
// results (GetOrZero) rather than for persistence — persistence goes
// through administrator.go's getOrCreate/ApplyResult, which build Entity
// values directly.
type Builder struct {
	tenantId    uuid.UUID
	id          uuid.UUID
	characterId uint32
	gameType    GameType
	wins        uint32
	ties        uint32
	losses      uint32
}

func NewBuilder(tenantId uuid.UUID, characterId uint32, gameType GameType) *Builder {
	return &Builder{
		tenantId:    tenantId,
		characterId: characterId,
		gameType:    gameType,
	}
}

func (b *Builder) SetId(id uuid.UUID) *Builder {
	b.id = id
	return b
}

func (b *Builder) SetWins(wins uint32) *Builder {
	b.wins = wins
	return b
}

func (b *Builder) SetTies(ties uint32) *Builder {
	b.ties = ties
	return b
}

func (b *Builder) SetLosses(losses uint32) *Builder {
	b.losses = losses
	return b
}

func (b *Builder) Build() (Model, error) {
	if b.tenantId == uuid.Nil {
		return Model{}, errors.New("tenantId is required")
	}
	if b.characterId == 0 {
		return Model{}, errors.New("characterId is required")
	}
	if b.gameType == "" {
		return Model{}, errors.New("gameType is required")
	}

	return Model{
		tenantId:    b.tenantId,
		id:          b.id,
		characterId: b.characterId,
		gameType:    b.gameType,
		wins:        b.wins,
		ties:        b.ties,
		losses:      b.losses,
	}, nil
}
