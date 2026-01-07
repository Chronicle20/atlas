package system_message

import (
	"atlas-saga-orchestrator/kafka/message/system_message"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

// SendMessageCommandProvider creates a Kafka message for sending system messages to a character
func SendMessageCommandProvider(transactionId uuid.UUID, worldId byte, channelId byte, characterId uint32, messageType string, message string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &system_message.Command[system_message.SendMessageBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		ChannelId:     channelId,
		CharacterId:   characterId,
		Type:          system_message.CommandSendMessage,
		Body: system_message.SendMessageBody{
			MessageType: messageType,
			Message:     message,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
