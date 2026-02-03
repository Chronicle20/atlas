package character

import (
	character2 "atlas-effective-stats/kafka/message/character"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func clampHPCommandProvider(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, characterId uint32, maxValue uint16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &character2.Command[character2.ClampHPBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		CharacterId:   characterId,
		Type:          character2.CommandClampHP,
		Body: character2.ClampHPBody{
			ChannelId: channelId,
			MaxValue:  maxValue,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func clampMPCommandProvider(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, characterId uint32, maxValue uint16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &character2.Command[character2.ClampMPBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		CharacterId:   characterId,
		Type:          character2.CommandClampMP,
		Body: character2.ClampMPBody{
			ChannelId: channelId,
			MaxValue:  maxValue,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
