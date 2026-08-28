package note

import (
	note2 "atlas-saga-orchestrator/kafka/message/note"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// CreateNoteCommandProvider builds the note CREATE command carrying the saga
// transaction id. CharacterId is the RECEIVER (atlas-notes stores notes keyed
// by the receiving character).
func CreateNoteCommandProvider(transactionId uuid.UUID, receiverId uint32, senderId uint32, message string, flag byte, giftNote bool) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(receiverId))
	value := &note2.Command[note2.CommandCreateBody]{
		TransactionId: transactionId,
		CharacterId:   receiverId,
		Type:          note2.CommandTypeCreate,
		Body: note2.CommandCreateBody{
			SenderId: senderId,
			Message:  message,
			Flag:     flag,
			GiftNote: giftNote,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
