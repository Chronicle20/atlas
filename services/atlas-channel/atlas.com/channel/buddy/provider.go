package buddy

import (
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func requestAddBuddyCommandProvider(characterId uint32, worldId byte, targetId uint32, targetName string, group string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &command[requestAddBuddyCommandBody]{
		WorldId:     worldId,
		CharacterId: characterId,
		Type:        CommandTypeRequestAdd,
		Body: requestAddBuddyCommandBody{
			CharacterId:   targetId,
			CharacterName: targetName,
			Group:         group,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func requestDeleteBuddyCommandProvider(characterId uint32, worldId byte, targetId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &command[requestDeleteBuddyCommandBody]{
		WorldId:     worldId,
		CharacterId: characterId,
		Type:        CommandTypeRequestDelete,
		Body: requestDeleteBuddyCommandBody{
			CharacterId: targetId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
