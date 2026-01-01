package pet

import (
	pet2 "atlas-saga-orchestrator/kafka/message/pet"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func GainClosenessProvider(petId uint32, amount uint16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(petId))
	value := &pet2.Command[pet2.GainClosenessCommandBody]{
		PetId: petId,
		Type:  pet2.CommandTypeGainCloseness,
		Body: pet2.GainClosenessCommandBody{
			Amount: amount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
