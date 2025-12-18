package expression

import (
	expression2 "atlas-channel/kafka/message/expression"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func SetCommandProvider(characterId uint32, m _map.Model, expression uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &expression2.Command{
		CharacterId: characterId,
		WorldId:     m.WorldId(),
		ChannelId:   m.ChannelId(),
		MapId:       m.MapId(),
		Expression:  expression,
	}
	return producer.SingleMessageProvider(key, value)
}
