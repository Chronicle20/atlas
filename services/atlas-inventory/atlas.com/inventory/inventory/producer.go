package inventory

import (
	"atlas-inventory/kafka/message/inventory"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func CreatedEventStatusProvider(characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &inventory.StatusEvent[inventory.CreatedStatusEventBody]{
		CharacterId: characterId,
		Type:        inventory.StatusEventTypeCreated,
		Body:        inventory.CreatedStatusEventBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

func DeletedEventStatusProvider(characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &inventory.StatusEvent[inventory.DeletedStatusEventBody]{
		CharacterId: characterId,
		Type:        inventory.StatusEventTypeDeleted,
		Body:        inventory.DeletedStatusEventBody{},
	}
	return producer.SingleMessageProvider(key, value)
}
