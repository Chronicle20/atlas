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
