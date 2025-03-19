package consumable

import (
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func requestItemConsumeCommandProvider(characterId uint32, source int16, itemId uint32, quantity int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &command[requestItemConsumeBody]{
		CharacterId: characterId,
		Type:        CommandRequestItemConsume,
		Body: requestItemConsumeBody{
			Source:   source,
			ItemId:   itemId,
			Quantity: quantity,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func requestScrollCommandProvider(characterId uint32, scrollSlot int16, equipScroll int16, whiteScroll bool, legendarySpirit bool) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &command[requestScrollBody]{
		CharacterId: characterId,
		Type:        CommandRequestScroll,
		Body: requestScrollBody{
			ScrollSlot:      scrollSlot,
			EquipSlot:       equipScroll,
			WhiteScroll:     whiteScroll,
			LegendarySpirit: legendarySpirit,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
