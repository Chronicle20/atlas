package buddylist

import (
	buddylist2 "atlas-saga-orchestrator/kafka/message/buddylist"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func RequestDeleteProvider(transactionId uuid.UUID, characterId character.Id, worldId world.Id, targetId character.Id) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &buddylist2.Command[buddylist2.RequestDeleteBuddyCommandBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		CharacterId:   characterId,
		Type:          buddylist2.CommandTypeRequestDelete,
		Body: buddylist2.RequestDeleteBuddyCommandBody{
			CharacterId: targetId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// RestoreProvider builds the RESTORE command that undoes one direction of a
// severance: it puts targetId back on characterId's buddy list (task-227
// FR-4.8). Keyed on characterId, exactly like RequestDeleteProvider, so the
// restore lands on the same partition as the delete it inverts.
func RestoreProvider(transactionId uuid.UUID, characterId character.Id, worldId world.Id, targetId character.Id) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &buddylist2.Command[buddylist2.RestoreBuddyCommandBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		CharacterId:   characterId,
		Type:          buddylist2.CommandTypeRestore,
		Body: buddylist2.RestoreBuddyCommandBody{
			CharacterId: targetId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func IncreaseCapacityProvider(transactionId uuid.UUID, characterId character.Id, worldId world.Id, newCapacity byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &buddylist2.Command[buddylist2.IncreaseCapacityCommandBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		CharacterId:   characterId,
		Type:          buddylist2.CommandTypeIncreaseCapacity,
		Body: buddylist2.IncreaseCapacityCommandBody{
			NewCapacity: newCapacity,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
