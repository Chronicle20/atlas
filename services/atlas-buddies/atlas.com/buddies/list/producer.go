package list

import (
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func createCommandProvider(characterId uint32, capacity byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &command[createCommandBody]{
		CharacterId: characterId,
		Type:        CommandTypeCreate,
		Body: createCommandBody{
			Capacity: capacity,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func errorStatusEventProvider(characterId uint32, worldId byte, error string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &statusEvent[errorStatusEventBody]{
		CharacterId: characterId,
		WorldId:     worldId,
		Type:        StatusEventTypeError,
		Body: errorStatusEventBody{
			Error: error,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
