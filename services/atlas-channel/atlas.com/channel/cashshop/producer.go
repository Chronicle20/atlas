package cashshop

import (
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func characterEnterCashShopStatusEventProvider(actorId uint32, worldId byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(actorId))
	value := &statusEvent[characterMovementBody]{
		WorldId: worldId,
		Type:    EventCashShopStatusTypeCharacterEnter,
		Body: characterMovementBody{
			CharacterId: actorId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func characterExitCashShopStatusEventProvider(actorId uint32, worldId byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(actorId))
	value := &statusEvent[characterMovementBody]{
		WorldId: worldId,
		Type:    EventCashShopStatusTypeCharacterExit,
		Body: characterMovementBody{
			CharacterId: actorId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
