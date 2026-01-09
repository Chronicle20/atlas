package buddylist

import (
	buddylist2 "atlas-saga-orchestrator/kafka/message/buddylist"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func IncreaseCapacityProvider(transactionId uuid.UUID, characterId uint32, worldId byte, newCapacity byte) model.Provider[[]kafka.Message] {
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
