package wishlist

import (
	"atlas-cashshop/kafka/message/wishlist"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func AddStatusEventProvider(characterId uint32, serialNumber uint32, itemId uuid.UUID) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &wishlist.StatusEvent[wishlist.StatusEventAddedBody]{
		CharacterId: characterId,
		Type:        wishlist.StatusEventTypeAdded,
		Body: wishlist.StatusEventAddedBody{
			SerialNumber: serialNumber,
			ItemId:       itemId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func DeleteStatusEventProvider(characterId uint32, itemId uuid.UUID) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &wishlist.StatusEvent[wishlist.StatusEventDeletedBody]{
		CharacterId: characterId,
		Type:        wishlist.StatusEventTypeDeleted,
		Body: wishlist.StatusEventDeletedBody{
			ItemId: itemId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func DeleteAllStatusEventProvider(characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &wishlist.StatusEvent[wishlist.StatusEventDeletedAllBody]{
		CharacterId: characterId,
		Type:        wishlist.StatusEventTypeDeletedAll,
		Body:        wishlist.StatusEventDeletedAllBody{},
	}
	return producer.SingleMessageProvider(key, value)
}