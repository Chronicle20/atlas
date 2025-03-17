package pet

import (
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func awardFullnessCommandProvider(actorId uint32, petId uint64, amount byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(petId))
	value := &command[awardFullnessCommandBody]{
		ActorId: actorId,
		PetId:   petId,
		Type:    CommandAwardFullness,
		Body: awardFullnessCommandBody{
			Amount: amount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
