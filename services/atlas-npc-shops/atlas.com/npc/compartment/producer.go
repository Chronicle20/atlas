package compartment

import (
	"atlas-npc/kafka/message/compartment"
	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
	"time"
)

func RequestCreateAssetCommandProvider(characterId uint32, inventoryType inventory.Type, templateId uint32, quantity uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &compartment.Command[compartment.CreateAssetCommandBody]{
		CharacterId:   characterId,
		InventoryType: byte(inventoryType),
		Type:          compartment.CommandCreateAsset,
		Body: compartment.CreateAssetCommandBody{
			TemplateId:   templateId,
			Quantity:     quantity,
			Expiration:   time.Time{},
			OwnerId:      0,
			Flag:         0,
			Rechargeable: 0,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
