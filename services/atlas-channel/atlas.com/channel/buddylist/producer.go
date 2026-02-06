package buddylist

import (
	"atlas-channel/kafka/message/buddylist"

	"github.com/Chronicle20/atlas-constants/character"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func RequestAddBuddyCommandProvider(characterId character.Id, worldId world.Id, targetId character.Id, group string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &buddylist.Command[buddylist.RequestAddBuddyCommandBody]{
		WorldId:     worldId,
		CharacterId: characterId,
		Type:        buddylist.CommandTypeRequestAdd,
		Body: buddylist.RequestAddBuddyCommandBody{
			CharacterId: targetId,
			Group:       group,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func RequestDeleteBuddyCommandProvider(characterId character.Id, worldId world.Id, targetId character.Id) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &buddylist.Command[buddylist.RequestDeleteBuddyCommandBody]{
		WorldId:     worldId,
		CharacterId: characterId,
		Type:        buddylist.CommandTypeRequestDelete,
		Body: buddylist.RequestDeleteBuddyCommandBody{
			CharacterId: targetId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
