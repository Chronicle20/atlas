package expression

import (
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func expressionCommandProvider(characterId uint32, m _map.Model, expression uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &expressionCommand{
		CharacterId: characterId,
		WorldId:     byte(m.WorldId()),
		ChannelId:   byte(m.ChannelId()),
		MapId:       uint32(m.MapId()),
		Expression:  expression,
	}
	return producer.SingleMessageProvider(key, value)
}
