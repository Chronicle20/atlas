package food

import (
	"atlas-channel/kafka/message/food"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

// RequestFeedCommandProvider builds the taming-mob (mount) food command. worldId
// flows to consumables via the field-derived WorldId/ChannelId/MapId/Instance.
func RequestFeedCommandProvider(f field.Model, characterId character.Id, slot int16, itemId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &food.Command[food.RequestFeedBody]{
		WorldId:     f.WorldId(),
		ChannelId:   f.ChannelId(),
		MapId:       f.MapId(),
		Instance:    f.Instance(),
		CharacterId: characterId,
		Type:        food.CommandRequestFeed,
		Body: food.RequestFeedBody{
			Slot:   slot,
			ItemId: itemId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
