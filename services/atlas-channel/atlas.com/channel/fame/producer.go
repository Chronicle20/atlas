package fame

import (
	fame2 "atlas-channel/kafka/message/fame"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func RequestChangeFameCommandProvider(f field.Model, characterId uint32, targetId uint32, amount int8) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &fame2.Command[fame2.RequestChangeCommandBody]{
		WorldId:     f.WorldId(),
		CharacterId: characterId,
		Type:        fame2.CommandTypeRequestChange,
		Body: fame2.RequestChangeCommandBody{
			ChannelId: f.ChannelId(),
			MapId:     f.MapId(),
			Instance:  f.Instance(),
			TargetId:  targetId,
			Amount:    amount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
