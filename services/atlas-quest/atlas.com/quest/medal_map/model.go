package medal_map

import (
	"github.com/google/uuid"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

// Model is one visited-map record for a character's quest.
type Model struct {
	id          uuid.UUID
	characterId uint32
	questId     uint32
	mapId       _map.Id
}

func (m Model) Id() uuid.UUID {
	return m.id
}

func (m Model) CharacterId() uint32 {
	return m.characterId
}

func (m Model) QuestId() uint32 {
	return m.questId
}

func (m Model) MapId() _map.Id {
	return m.mapId
}

// RecordResult is the outcome of recording a visited map: the resulting
// distinct-map count for the character's quest, and whether this particular
// map was newly recorded (false when it was already present -- Cosmic's
// qs.addMedalMap dedup).
type RecordResult struct {
	Count         uint32
	NewlyRecorded bool
}
