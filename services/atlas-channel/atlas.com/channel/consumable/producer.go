package consumable

import (
	"atlas-channel/kafka/message/consumable"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func RequestItemConsumeCommandProvider(characterId uint32, source int16, itemId uint32, quantity int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &consumable.Command[consumable.RequestItemConsumeBody]{
		CharacterId: characterId,
		Type:        consumable.CommandRequestItemConsume,
		Body: consumable.RequestItemConsumeBody{
			Source:   source,
			ItemId:   itemId,
			Quantity: quantity,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func RequestScrollCommandProvider(characterId uint32, scrollSlot int16, equipScroll int16, whiteScroll bool, legendarySpirit bool) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &consumable.Command[consumable.RequestScrollBody]{
		CharacterId: characterId,
		Type:        consumable.CommandRequestScroll,
		Body: consumable.RequestScrollBody{
			ScrollSlot:      scrollSlot,
			EquipSlot:       equipScroll,
			WhiteScroll:     whiteScroll,
			LegendarySpirit: legendarySpirit,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
