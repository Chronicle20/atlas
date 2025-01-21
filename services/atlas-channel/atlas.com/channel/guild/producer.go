package guild

import (
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func requestCreateProvider(worldId byte, channelId byte, mapId uint32, characterId uint32, name string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &command[requestCreateBody]{
		CharacterId: characterId,
		Type:        CommandTypeRequestCreate,
		Body: requestCreateBody{
			WorldId:   worldId,
			ChannelId: channelId,
			MapId:     mapId,
			Name:      name,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
