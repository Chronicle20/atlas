package character

import (
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func changeHPCommandProvider(characterId uint32, amount int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &command[changeHPCommandBody]{
		CharacterId: characterId,
		Type:        CommandChangeHP,
		Body: changeHPCommandBody{
			Amount: amount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func changeMPCommandProvider(characterId uint32, amount int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &command[changeMPCommandBody]{
		CharacterId: characterId,
		Type:        CommandChangeMP,
		Body: changeMPCommandBody{
			Amount: amount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
