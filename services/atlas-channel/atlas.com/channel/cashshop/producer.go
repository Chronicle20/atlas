package cashshop

import (
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func characterEnterCashShopStatusEventProvider(actorId uint32, m _map.Model) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(actorId))
	value := &statusEvent[characterMovementBody]{
		WorldId: byte(m.WorldId()),
		Type:    EventCashShopStatusTypeCharacterEnter,
		Body: characterMovementBody{
			CharacterId: actorId,
			ChannelId:   byte(m.ChannelId()),
			MapId:       uint32(m.MapId()),
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func characterExitCashShopStatusEventProvider(actorId uint32, m _map.Model) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(actorId))
	value := &statusEvent[characterMovementBody]{
		WorldId: byte(m.WorldId()),
		Type:    EventCashShopStatusTypeCharacterExit,
		Body: characterMovementBody{
			CharacterId: actorId,
			ChannelId:   byte(m.ChannelId()),
			MapId:       uint32(m.MapId()),
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func RequestInventoryIncreaseByTypeCommandProvider(characterId uint32, currency uint32, inventoryType byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &Command[RequestInventoryIncreaseByTypeCommandBody]{
		CharacterId: characterId,
		Type:        CommandTypeRequestInventoryIncreaseByType,
		Body: RequestInventoryIncreaseByTypeCommandBody{
			Currency:      currency,
			InventoryType: inventoryType,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func RequestInventoryIncreaseByItemCommandProvider(characterId uint32, currency uint32, serialNumber uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &Command[RequestInventoryIncreaseByItemCommandBody]{
		CharacterId: characterId,
		Type:        CommandTypeRequestInventoryIncreaseByItem,
		Body: RequestInventoryIncreaseByItemCommandBody{
			Currency:     currency,
			SerialNumber: serialNumber,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func RequestStorageIncreaseCommandProvider(characterId uint32, currency uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &Command[RequestStorageIncreaseBody]{
		CharacterId: characterId,
		Type:        CommandTypeRequestStorageIncrease,
		Body: RequestStorageIncreaseBody{
			Currency: currency,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func RequestStorageIncreaseByItemCommandProvider(characterId uint32, currency uint32, serialNumber uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &Command[RequestStorageIncreaseByItemCommandBody]{
		CharacterId: characterId,
		Type:        CommandTypeRequestStorageIncreaseByItem,
		Body: RequestStorageIncreaseByItemCommandBody{
			Currency:     currency,
			SerialNumber: serialNumber,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func RequestCharacterSlotIncreaseByItemCommandProvider(characterId uint32, currency uint32, serialNumber uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &Command[RequestCharacterSlotIncreaseByItemCommandBody]{
		CharacterId: characterId,
		Type:        CommandTypeRequestCharacterSlotIncreaseByItem,
		Body: RequestCharacterSlotIncreaseByItemCommandBody{
			Currency:     currency,
			SerialNumber: serialNumber,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
