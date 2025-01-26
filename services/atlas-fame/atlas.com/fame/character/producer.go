package character

import (
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func requestChangeFameCommandProvider(characterId uint32, worldId byte, actorId uint32, amount int8) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &commandEvent[requestChangeFameBody]{
		CharacterId: characterId,
		WorldId:     worldId,
		Type:        CommandRequestChangeFame,
		Body: requestChangeFameBody{
			ActorId:   actorId,
			ActorType: CommandActorTypeCharacter,
			Amount:    amount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
