package fame

import (
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func requestChangeFameCommandProvider(worldId byte, channelId byte, characterId uint32, mapId uint32, targetId uint32, amount int8) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &command[requestChangeCommandBody]{
		WorldId:     worldId,
		CharacterId: characterId,
		Type:        CommandTypeRequestChange,
		Body: requestChangeCommandBody{
			ChannelId: channelId,
			MapId:     mapId,
			TargetId:  targetId,
			Amount:    amount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
