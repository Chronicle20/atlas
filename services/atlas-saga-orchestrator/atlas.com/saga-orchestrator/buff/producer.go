package buff

import (
	buffMsg "atlas-saga-orchestrator/kafka/message/buff"

	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

// CancelAllCommandProvider creates a Kafka message for canceling all buffs on a character
func CancelAllCommandProvider(worldId byte, channelId byte, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &buffMsg.Command[buffMsg.CancelAllBody]{
		WorldId:     worldId,
		ChannelId:   channelId,
		CharacterId: characterId,
		Type:        buffMsg.CommandTypeCancelAll,
		Body:        buffMsg.CancelAllBody{},
	}
	return producer.SingleMessageProvider(key, value)
}
