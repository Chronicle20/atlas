package pet

import (
	message "atlas-consumables/kafka/message/pet"

	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func awardFullnessCommandProvider(actorId uint32, petId uint64, amount byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(petId))
	value := &message.Command[message.AwardFullnessCommandBody]{
		ActorId: actorId,
		PetId:   petId,
		Type:    message.CommandAwardFullness,
		Body: message.AwardFullnessCommandBody{
			Amount: amount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
