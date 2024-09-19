package message

import (
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func generalChatEventProvider(worldId byte, channelId byte, mapId uint32, characterId uint32, message string, balloonOnly bool) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := generalChatEvent{
		WorldId:     worldId,
		ChannelId:   channelId,
		MapId:       mapId,
		CharacterId: characterId,
		Message:     message,
		BalloonOnly: balloonOnly,
	}
	return producer.SingleMessageProvider(key, value)
}
