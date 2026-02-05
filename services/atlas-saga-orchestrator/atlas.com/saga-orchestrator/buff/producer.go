package buff

import (
	buffMsg "atlas-saga-orchestrator/kafka/message/buff"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

// CancelAllCommandProvider creates a Kafka message for canceling all buffs on a character
func CancelAllCommandProvider(field field.Model, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &buffMsg.Command[buffMsg.CancelAllBody]{
		WorldId:     field.WorldId(),
		ChannelId:   field.ChannelId(),
		MapId:       field.MapId(),
		Instance:    field.Instance(),
		CharacterId: characterId,
		Type:        buffMsg.CommandTypeCancelAll,
		Body:        buffMsg.CancelAllBody{},
	}
	return producer.SingleMessageProvider(key, value)
}
