package inventory

import (
	"atlas-character/tenant"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func equipItemCommandProvider(tenant tenant.Model, characterId uint32, source int16, destination int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &equipItemCommand{
		Tenant:      tenant,
		CharacterId: characterId,
		Source:      source,
		Destination: destination,
	}
	return producer.SingleMessageProvider(key, value)
}

func unequipItemCommandProvider(tenant tenant.Model, characterId uint32, source int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &unequipItemCommand{
		Tenant:      tenant,
		CharacterId: characterId,
		Source:      source,
	}
	return producer.SingleMessageProvider(key, value)
}

func itemGainEventProvider(tenant tenant.Model, characterId uint32, itemId uint32, quantity uint32, slot int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &gainItemEvent{
		Tenant:      tenant,
		CharacterId: characterId,
		ItemId:      itemId,
		Quantity:    quantity,
		Slot:        slot,
	}
	return producer.SingleMessageProvider(key, value)
}

func itemEquippedProvider(tenant tenant.Model, characterId uint32, itemId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &equipChangedEvent{
		Tenant:      tenant,
		CharacterId: characterId,
		Change:      "EQUIPPED",
		ItemId:      itemId,
	}
	return producer.SingleMessageProvider(key, value)
}

func itemUnequippedProvider(tenant tenant.Model, characterId uint32, itemId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &equipChangedEvent{
		Tenant:      tenant,
		CharacterId: characterId,
		Change:      "UNEQUIPPED",
		ItemId:      itemId,
	}
	return producer.SingleMessageProvider(key, value)
}
