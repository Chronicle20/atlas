package consumable

import (
	"atlas-consumables/kafka/message/consumable"
	foodmsg "atlas-consumables/kafka/message/food"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func ErrorEventProvider(characterId character.Id, error string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &consumable.Event[consumable.ErrorBody]{
		CharacterId: characterId,
		Type:        consumable.EventTypeError,
		Body: consumable.ErrorBody{
			Error: error,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func ScrollEventProvider(characterId character.Id) func(success bool, cursed bool, legendarySpirit bool, whiteScroll bool) model.Provider[[]kafka.Message] {
	return func(success bool, cursed bool, legendarySpirit bool, whiteScroll bool) model.Provider[[]kafka.Message] {
		key := producer.CreateKey(int(characterId))
		value := &consumable.Event[consumable.ScrollBody]{
			CharacterId: characterId,
			Type:        consumable.EventTypeScroll,
			Body: consumable.ScrollBody{
				Success:         success,
				Cursed:          cursed,
				LegendarySpirit: legendarySpirit,
				WhiteScroll:     whiteScroll,
			},
		}
		return producer.SingleMessageProvider(key, value)
	}
}

func ViciousHammerEventProvider(characterId character.Id, success bool, reason ViciousHammerReason) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &consumable.Event[consumable.ViciousHammerBody]{
		CharacterId: characterId,
		Type:        consumable.EventTypeViciousHammer,
		Body: consumable.ViciousHammerBody{
			Success: success,
			Reason:  string(reason),
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// TamingMobFedEventProvider builds the TamingMobFed event emitted after a
// revitalizer (classification 226) is consumed. Keyed by characterId so
// atlas-mounts processes a character's feeds in order. tirednessHeal is the
// pinned server constant foodmsg.RevitalizerTirednessHeal.
func TamingMobFedEventProvider(worldId world.Id, characterId uint32, itemId uint32, tirednessHeal int32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &foodmsg.Event{
		WorldId:       worldId,
		CharacterId:   characterId,
		ItemId:        itemId,
		TirednessHeal: tirednessHeal,
	}
	return producer.SingleMessageProvider(key, value)
}

func EffectAppliedEventProvider(characterId character.Id, itemId item.Id, transactionId uuid.UUID) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &consumable.Event[consumable.EffectAppliedBody]{
		CharacterId: characterId,
		Type:        consumable.EventTypeEffectApplied,
		Body: consumable.EffectAppliedBody{
			ItemId:        itemId,
			TransactionId: transactionId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
