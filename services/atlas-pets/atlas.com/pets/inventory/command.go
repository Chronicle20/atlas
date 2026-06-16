package inventory

import (
	"atlas-pets/kafka/message"
	compartmentmsg "atlas-pets/kafka/message/compartment"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
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
