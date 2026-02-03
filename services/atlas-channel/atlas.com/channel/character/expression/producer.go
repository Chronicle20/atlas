package expression

import (
	expression2 "atlas-channel/kafka/message/expression"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func SetCommandProvider(characterId uint32, f field.Model, expression uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &expression2.Command{
		CharacterId: characterId,
		WorldId:     f.WorldId(),
		ChannelId:   f.ChannelId(),
		MapId:       f.MapId(),
		Instance:    f.Instance(),
		Expression:  expression,
	}
	return producer.SingleMessageProvider(key, value)
}
