package inventory

import (
	compartmentmsg "atlas-pets/kafka/message/compartment"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func changeTemplateCommandProvider(transactionId uuid.UUID, characterId uint32, petId uint32, newTemplateId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &compartmentmsg.Command[compartmentmsg.ChangeTemplateCommandBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		InventoryType: byte(inventory.TypeValueCash),
		Type:          compartmentmsg.CommandChangeTemplate,
		Body: compartmentmsg.ChangeTemplateCommandBody{
			PetId:         petId,
			NewTemplateId: newTemplateId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func resetPetExpirationCommandProvider(transactionId uuid.UUID, characterId uint32, petId uint32, expiration time.Time, sourceTemplateId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &compartmentmsg.Command[compartmentmsg.ResetPetExpirationCommandBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		InventoryType: byte(inventory.TypeValueCash),
		Type:          compartmentmsg.CommandResetPetExpiration,
		Body: compartmentmsg.ResetPetExpirationCommandBody{
			PetId:            petId,
			Expiration:       expiration,
			SourceTemplateId: sourceTemplateId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
