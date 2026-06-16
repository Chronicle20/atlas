package pet

import (
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

// AwardClosenessCommandProvider builds an additive AWARD_CLOSENESS command for a pet.
func AwardClosenessCommandProvider(petId uint32, amount uint16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(petId))
	value := &Command[AwardClosenessCommandBody]{
		TransactionId: uuid.New(),
		PetId:         petId,
		Type:          CommandAwardCloseness,
		Body: AwardClosenessCommandBody{
			Amount: amount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
