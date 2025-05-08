package shops

import (
	"atlas-npc/kafka/message/shops"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func enteredEventProvider(characterId uint32, npcId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &shops.StatusEvent[shops.StatusEventEnteredBody]{
		CharacterId: characterId,
		Type:        shops.StatusEventTypeEntered,
		Body: shops.StatusEventEnteredBody{
			NpcTemplateId: npcId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func exitedEventProvider(characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &shops.StatusEvent[shops.StatusEventExitedBody]{
		CharacterId: characterId,
		Type:        shops.StatusEventTypeExited,
		Body:        shops.StatusEventExitedBody{},
	}
	return producer.SingleMessageProvider(key, value)
}
