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