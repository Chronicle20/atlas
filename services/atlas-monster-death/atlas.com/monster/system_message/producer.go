package system_message

import (
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// ShowHintCommandProvider creates a Kafka message for showing a hint box
func ShowHintCommandProvider(transactionId uuid.UUID, ch channel.Model, characterId uint32, hint string, width uint16, height uint16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &Command[ShowHintBody]{
		TransactionId: transactionId,
		WorldId:       ch.WorldId(),
		ChannelId:     ch.Id(),
		CharacterId:   characterId,
		Type:          CommandShowHint,
		Body: ShowHintBody{
			Hint:   hint,
			Width:  width,
			Height: height,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
