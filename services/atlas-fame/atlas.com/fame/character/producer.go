package character

import (
	messageCharacter "atlas-fame/kafka/message/character"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func requestChangeFameCommandProvider(transactionId uuid.UUID, characterId uint32, worldId world.Id, actorId uint32, amount int8) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &messageCharacter.CommandEvent[messageCharacter.RequestChangeFameBody]{
		TransactionId: transactionId,
		CharacterId: characterId,
		WorldId:     worldId,
		Type:        messageCharacter.CommandRequestChangeFame,
		Body: messageCharacter.RequestChangeFameBody{
			ActorId:   actorId,
			ActorType: messageCharacter.CommandActorTypeCharacter,
			Amount:    amount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
