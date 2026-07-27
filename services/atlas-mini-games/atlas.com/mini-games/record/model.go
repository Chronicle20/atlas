package record

import "github.com/google/uuid"

// GameType identifies which mini-game a game_records row tracks.
type GameType string

const (
	GameTypeOmok       GameType = "OMOK"
	GameTypeMatchCards GameType = "MATCH_CARDS"
)

// AllGameTypes enumerates every GameType a character has a record for.
// GetByCharacter iterates this so its result is always zero-filled for
// every mini-game, even when the character has no rows yet.
var AllGameTypes = []GameType{GameTypeOmok, GameTypeMatchCards}

// Model is the immutable win/tie/loss record for one character and game
// type. A Model with a zero Id (never persisted) represents "no rows yet"
// and is returned by GetOrZero/GetByCharacter for an absent row.
type Model struct {
	tenantId    uuid.UUID
	id          uuid.UUID
	characterId uint32
	gameType    GameType
	wins        uint32
	ties        uint32
	losses      uint32
}

func (m Model) Id() uuid.UUID {
	return m.id
}

func (m Model) TenantId() uuid.UUID {
	return m.tenantId
}

func (m Model) CharacterId() uint32 {
	return m.characterId
}

func (m Model) GameType() GameType {
	return m.gameType
}

func (m Model) Wins() uint32 {
	return m.wins
}

func (m Model) Ties() uint32 {
	return m.ties
}

func (m Model) Losses() uint32 {
	return m.losses
}
