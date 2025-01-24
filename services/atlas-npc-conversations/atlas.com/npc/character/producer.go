package character

import (
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func requestChangeMesoCommandProvider(characterId uint32, worldId byte, amount int32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &commandEvent[requestChangeMesoBody]{
		CharacterId: characterId,
		WorldId:     worldId,
		Type:        CommandRequestChangeMeso,
		Body: requestChangeMesoBody{
			Amount: amount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
