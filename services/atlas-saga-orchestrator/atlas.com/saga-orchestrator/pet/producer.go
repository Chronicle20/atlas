package pet

import (
	pet2 "atlas-saga-orchestrator/kafka/message/pet"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
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

func EvolveProvider(transactionId uuid.UUID, petId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(petId))
	value := &pet2.Command[pet2.EvolveCommandBody]{
		TransactionId: transactionId,
		PetId:         petId,
		Type:          pet2.CommandPetEvolve,
		Body:          pet2.EvolveCommandBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

func RenameProvider(transactionId uuid.UUID, petId uint32, characterId uint32, name string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(petId))
	value := &pet2.Command[pet2.RenameCommandBody]{
		TransactionId: transactionId,
		ActorId:       characterId,
		PetId:         petId,
		Type:          pet2.CommandPetRename,
		Body: pet2.RenameCommandBody{
			Name: name,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
