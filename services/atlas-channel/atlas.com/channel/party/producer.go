package party

import (
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func createCommandProvider(actorId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(actorId))
	value := &commandEvent[createCommandBody]{
		ActorId: actorId,
		Type:    CommandPartyCreate,
		Body:    createCommandBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

func leaveCommandProvider(actorId uint32, partyId uint32, force bool) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(actorId))
	value := &commandEvent[leaveCommandBody]{
		ActorId: actorId,
		Type:    CommandPartyLeave,
		Body: leaveCommandBody{
			PartyId: partyId,
			Force:   force,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func changeLeaderCommandProvider(actorId uint32, partyId uint32, leaderId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(actorId))
	value := &commandEvent[changeLeaderBody]{
		ActorId: actorId,
		Type:    CommandPartyChangeLeader,
		Body: changeLeaderBody{
			LeaderId: leaderId,
			PartyId:  partyId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func requestInviteCommandProvider(actorId uint32, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(actorId))
	value := &commandEvent[requestInviteBody]{
		ActorId: actorId,
		Type:    CommandPartyRequestInvite,
		Body: requestInviteBody{
			CharacterId: characterId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
