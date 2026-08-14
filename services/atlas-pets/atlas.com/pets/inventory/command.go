package inventory

import (
	"atlas-pets/kafka/message"
	compartmentmsg "atlas-pets/kafka/message/compartment"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// ChangeTemplate buffers a CHANGE_TEMPLATE command to atlas-inventory.
func (p *ProcessorImpl) ChangeTemplate(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, petId uint32, newTemplateId uint32) error {
	return func(transactionId uuid.UUID, characterId uint32, petId uint32, newTemplateId uint32) error {
		return mb.Put(compartmentmsg.EnvCommandTopic, changeTemplateCommandProvider(transactionId, characterId, petId, newTemplateId))
	}
}

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

// ResetPetExpiration buffers a RESET_PET_EXPIRATION command to atlas-inventory.
// It is buffered inside the revive's own database transaction + outbox, so the
// pet row update and this cascade commit together or not at all — the pet
// record and the inventory slot cannot diverge (design §7.1).
func (p *ProcessorImpl) ResetPetExpiration(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, petId uint32, expiration time.Time, sourceTemplateId uint32) error {
	return func(transactionId uuid.UUID, characterId uint32, petId uint32, expiration time.Time, sourceTemplateId uint32) error {
		return mb.Put(compartmentmsg.EnvCommandTopic, resetPetExpirationCommandProvider(transactionId, characterId, petId, expiration, sourceTemplateId))
	}
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
