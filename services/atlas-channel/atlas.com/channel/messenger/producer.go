package messenger

import (
	messenger2 "atlas-channel/kafka/message/messenger"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func CreateCommandProvider(actorId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(actorId))
	value := &messenger2.Command[messenger2.CreateCommandBody]{
		ActorId: actorId,
		Type:    messenger2.CommandMessengerCreate,
		Body:    messenger2.CreateCommandBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

func LeaveCommandProvider(actorId uint32, messengerId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(actorId))
	value := &messenger2.Command[messenger2.LeaveCommandBody]{
		ActorId: actorId,
		Type:    messenger2.CommandMessengerLeave,
		Body: messenger2.LeaveCommandBody{
			MessengerId: messengerId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func RequestInviteCommandProvider(actorId uint32, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(actorId))
	value := &messenger2.Command[messenger2.RequestInviteBody]{
		ActorId: actorId,
		Type:    messenger2.CommandMessengerRequestInvite,
		Body: messenger2.RequestInviteBody{
			CharacterId: characterId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
