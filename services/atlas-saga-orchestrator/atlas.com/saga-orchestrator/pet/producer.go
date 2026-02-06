package pet

import (
	pet2 "atlas-saga-orchestrator/kafka/message/pet"

	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func AwardClosenessProvider(transactionId uuid.UUID, petId uint32, amount uint16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(petId))
	value := &pet2.Command[pet2.AwardClosenessCommandBody]{
		TransactionId: transactionId,
		PetId:         petId,
		Type:          pet2.CommandTypeAwardCloseness,
		Body: pet2.AwardClosenessCommandBody{
			Amount: amount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
