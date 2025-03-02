package messenger

import (
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func createCommandProvider(actorId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(actorId))
	value := &commandEvent[createCommandBody]{
		ActorId: actorId,
		Type:    CommandMessengerCreate,
		Body:    createCommandBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

func leaveCommandProvider(actorId uint32, messengerId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(actorId))
	value := &commandEvent[leaveCommandBody]{
		ActorId: actorId,
		Type:    CommandMessengerLeave,
		Body: leaveCommandBody{
			MessengerId: messengerId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func requestInviteCommandProvider(actorId uint32, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(actorId))
	value := &commandEvent[requestInviteBody]{
		ActorId: actorId,
		Type:    CommandMessengerRequestInvite,
		Body: requestInviteBody{
			CharacterId: characterId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
