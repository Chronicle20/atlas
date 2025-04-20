package compartment

import (
	"atlas-character-factory/kafka/message/compartment"
	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
	"time"
)

func CreateAssetCommandProvider(characterId uint32, inventoryType inventory.Type, templateId uint32, quantity uint32, expiration time.Time, ownerId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &compartment.Command[compartment.CreateAssetCommandBody]{
		CharacterId:   characterId,
		InventoryType: byte(inventoryType),
		Type:          compartment.CommandCreateAsset,
		Body: compartment.CreateAssetCommandBody{
			TemplateId:   templateId,
			Quantity:     quantity,
			Expiration:   expiration,
			OwnerId:      ownerId,
			Flag:         0,
			Rechargeable: 0,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func EquipAssetCommandProvider(characterId uint32, inventoryType inventory.Type, source int16, destination int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &compartment.Command[compartment.EquipCommandBody]{
		CharacterId:   characterId,
		InventoryType: byte(inventoryType),
		Type:          compartment.CommandEquip,
		Body: compartment.EquipCommandBody{
			Source:      source,
			Destination: destination,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
