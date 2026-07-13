package consumable

import (
	"atlas-channel/kafka/message/consumable"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func RequestItemConsumeCommandProvider(f field.Model, characterId character.Id, source slot.Position, itemId item.Id, quantity int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &consumable.Command[consumable.RequestItemConsumeBody]{
		WorldId:     f.WorldId(),
		ChannelId:   f.ChannelId(),
		MapId:       f.MapId(),
		Instance:    f.Instance(),
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

func RequestScrollCommandProvider(f field.Model, characterId character.Id, scrollSlot slot.Position, equipSlot slot.Position, whiteScroll bool, legendarySpirit bool) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &consumable.Command[consumable.RequestScrollBody]{
		WorldId:     f.WorldId(),
		ChannelId:   f.ChannelId(),
		MapId:       f.MapId(),
		Instance:    f.Instance(),
		CharacterId: characterId,
		Type:        consumable.CommandRequestScroll,
		Body: consumable.RequestScrollBody{
			ScrollSlot:      scrollSlot,
			EquipSlot:       equipSlot,
			WhiteScroll:     whiteScroll,
			LegendarySpirit: legendarySpirit,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func RequestVegaScrollCommandProvider(f field.Model, characterId character.Id, vegaSlot slot.Position, vegaItemId item.Id, scrollSlot slot.Position, equipSlot slot.Position) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &consumable.Command[consumable.RequestVegaScrollBody]{
		WorldId:     f.WorldId(),
		ChannelId:   f.ChannelId(),
		MapId:       f.MapId(),
		Instance:    f.Instance(),
		CharacterId: characterId,
		Type:        consumable.CommandRequestVegaScroll,
		Body: consumable.RequestVegaScrollBody{
			VegaSlot:   vegaSlot,
			VegaItemId: vegaItemId,
			ScrollSlot: scrollSlot,
			EquipSlot:  equipSlot,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func RequestViciousHammerCommandProvider(f field.Model, characterId character.Id, hammerSlot slot.Position, equipSlot slot.Position) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &consumable.Command[consumable.RequestViciousHammerBody]{
		WorldId:     f.WorldId(),
		ChannelId:   f.ChannelId(),
		MapId:       f.MapId(),
		Instance:    f.Instance(),
		CharacterId: characterId,
		Type:        consumable.CommandRequestViciousHammer,
		Body: consumable.RequestViciousHammerBody{
			HammerSlot: hammerSlot,
			EquipSlot:  equipSlot,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
