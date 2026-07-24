package quest

import (
	questmessage "atlas-quest/kafka/message/quest"
	"atlas-quest/kafka/message/saga"
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// EventEmitter defines the interface for emitting quest-related events
type EventEmitter interface {
	EmitQuestStarted(transactionId uuid.UUID, characterId uint32, worldId world.Id, questId uint32, progress string, items []questmessage.ItemReward) error
	EmitQuestCompleted(transactionId uuid.UUID, characterId uint32, worldId world.Id, questId uint32, completedAt time.Time, items []questmessage.ItemReward) error
	EmitQuestForfeited(transactionId uuid.UUID, characterId uint32, worldId world.Id, questId uint32) error
	EmitProgressUpdated(transactionId uuid.UUID, characterId uint32, worldId world.Id, questId uint32, infoNumber uint32, progress string) error
	EmitSaga(s saga.Saga) error
}
