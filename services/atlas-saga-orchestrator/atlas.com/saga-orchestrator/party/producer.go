package party

import (
	"atlas-saga-orchestrator/kafka/message/party"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func RequestLeaveProvider(transactionId uuid.UUID, characterId uint32, partyId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &party.Command[party.LeaveBody]{
		ActorId: characterId,
		Type:    party.CommandTypeLeave,
		Body: party.LeaveBody{
			PartyId:     partyId,
			Force:       false,
			CharacterId: characterId,
		},
		TransactionId: transactionId,
	}
	return producer.SingleMessageProvider(key, value)
}
