package fame

import (
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func errorEventStatusProvider(worldId byte, channelId byte, characterId uint32, error string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &statusEvent[statusEventErrorBody]{
		WorldId:     worldId,
		CharacterId: characterId,
		Type:        StatusEventTypeError,
		Body: statusEventErrorBody{
			ChannelId: channelId,
			Error:     error,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
