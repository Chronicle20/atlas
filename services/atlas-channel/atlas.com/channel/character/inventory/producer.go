package inventory

import (
	"atlas-channel/tenant"
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

func unequipItemCommandProvider(tenant tenant.Model, characterId uint32, source int16, destination int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &unequipItemCommand{
		Tenant:      tenant,
		CharacterId: characterId,
		Source:      source,
		Destination: destination,
	}
	return producer.SingleMessageProvider(key, value)
}

func moveItemCommandProvider(tenant tenant.Model, characterId uint32, inventoryType byte, source int16, destination int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &moveItemCommand{
		Tenant:        tenant,
		CharacterId:   characterId,
		InventoryType: inventoryType,
		Source:        source,
		Destination:   destination,
	}
	return producer.SingleMessageProvider(key, value)
}

func dropItemCommandProvider(tenant tenant.Model, characterId uint32, inventoryType byte, source int16, quantity int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &dropItemCommand{
		Tenant:        tenant,
		CharacterId:   characterId,
		InventoryType: inventoryType,
		Source:        source,
		Quantity:      quantity,
	}
	return producer.SingleMessageProvider(key, value)
}
