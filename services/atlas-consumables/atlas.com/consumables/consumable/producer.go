package consumable

import (
	"atlas-consumables/kafka/message/consumable"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func ErrorEventProvider(characterId uint32, error string) model.Provider[[]kafka.Message] {
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

func ScrollEventProvider(characterId uint32) func(success bool, cursed bool, legendarySpirit bool, whiteScroll bool) model.Provider[[]kafka.Message] {
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

func EffectAppliedEventProvider(characterId uint32, itemId uint32, transactionId uuid.UUID) model.Provider[[]kafka.Message] {
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
