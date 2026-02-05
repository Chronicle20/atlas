package fame

import (
	messageFame "atlas-fame/kafka/message/fame"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func errorEventStatusProvider(transactionId uuid.UUID, ch channel.Model, characterId uint32, error string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &messageFame.StatusEvent[messageFame.StatusEventErrorBody]{
		TransactionId: transactionId,
		WorldId:       ch.WorldId(),
		CharacterId:   characterId,
		Type:          messageFame.StatusEventTypeError,
		Body: messageFame.StatusEventErrorBody{
			ChannelId: ch.Id(),
			Error:     error,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
