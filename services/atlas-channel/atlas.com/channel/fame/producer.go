package fame

import (
	fame2 "atlas-channel/kafka/message/fame"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func RequestChangeFameCommandProvider(m _map.Model, characterId uint32, targetId uint32, amount int8) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &fame2.Command[fame2.RequestChangeCommandBody]{
		WorldId:     m.WorldId(),
		CharacterId: characterId,
		Type:        fame2.CommandTypeRequestChange,
		Body: fame2.RequestChangeCommandBody{
			ChannelId: m.ChannelId(),
			MapId:     m.MapId(),
			TargetId:  targetId,
			Amount:    amount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
