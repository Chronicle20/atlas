package buddylist

import (
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func requestAddBuddyCommandProvider(characterId uint32, worldId world.Id, targetId uint32, group string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &command[requestAddBuddyCommandBody]{
		WorldId:     byte(worldId),
		CharacterId: characterId,
		Type:        CommandTypeRequestAdd,
		Body: requestAddBuddyCommandBody{
			CharacterId: targetId,
			Group:       group,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func requestDeleteBuddyCommandProvider(characterId uint32, worldId world.Id, targetId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &command[requestDeleteBuddyCommandBody]{
		WorldId:     byte(worldId),
		CharacterId: characterId,
		Type:        CommandTypeRequestDelete,
		Body: requestDeleteBuddyCommandBody{
			CharacterId: targetId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
