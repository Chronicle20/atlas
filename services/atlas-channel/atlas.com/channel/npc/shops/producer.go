package shops

import (
	shops2 "atlas-channel/kafka/message/npc/shop"

	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func ShopEnterCommandProvider(characterId uint32, npcTemplateId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &shops2.Command[shops2.CommandShopEnterBody]{
		CharacterId: characterId,
		Type:        shops2.CommandShopEnter,
		Body: shops2.CommandShopEnterBody{
			NpcTemplateId: npcTemplateId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func ShopExitCommandProvider(characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &shops2.Command[shops2.CommandShopExitBody]{
		CharacterId: characterId,
		Type:        shops2.CommandShopExit,
		Body:        shops2.CommandShopExitBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

func ShopBuyCommandProvider(characterId uint32, slot uint16, itemTemplateId uint32, quantity uint32, discountPrice uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &shops2.Command[shops2.CommandShopBuyBody]{
		CharacterId: characterId,
		Type:        shops2.CommandShopBuy,
		Body: shops2.CommandShopBuyBody{
			Slot:           slot,
			ItemTemplateId: itemTemplateId,
			Quantity:       quantity,
			DiscountPrice:  discountPrice,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func ShopSellCommandProvider(characterId uint32, slot int16, itemTemplateId uint32, quantity uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &shops2.Command[shops2.CommandShopSellBody]{
		CharacterId: characterId,
		Type:        shops2.CommandShopSell,
		Body: shops2.CommandShopSellBody{
			Slot:           slot,
			ItemTemplateId: itemTemplateId,
			Quantity:       quantity,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func ShopRechargeCommandProvider(characterId uint32, slot uint16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &shops2.Command[shops2.CommandShopRechargeBody]{
		CharacterId: characterId,
		Type:        shops2.CommandShopRecharge,
		Body: shops2.CommandShopRechargeBody{
			Slot: slot,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
