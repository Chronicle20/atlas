package teleport_rock

import (
	teleportrock2 "atlas-character/kafka/message/teleportrock"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func listUpdatedEventProvider(transactionId uuid.UUID, worldId world.Id, characterId uint32, vip bool, registered bool, maps []_map.Id) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &teleportrock2.StatusEvent[teleportrock2.ListUpdatedStatusBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		CharacterId:   characterId,
		Type:          teleportrock2.StatusEventTypeListUpdated,
		Body: teleportrock2.ListUpdatedStatusBody{
			Vip:        vip,
			Registered: registered,
			Maps:       maps,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func errorEventProvider(transactionId uuid.UUID, worldId world.Id, characterId uint32, vip bool, reason string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &teleportrock2.StatusEvent[teleportrock2.ErrorStatusBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		CharacterId:   characterId,
		Type:          teleportrock2.StatusEventTypeError,
		Body: teleportrock2.ErrorStatusBody{
			Vip:    vip,
			Reason: reason,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
