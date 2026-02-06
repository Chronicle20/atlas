package party

import (
	party2 "atlas-channel/kafka/message/party"

	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func CreateCommandProvider(actorId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(actorId))
	value := &party2.Command[party2.CreateCommandBody]{
		ActorId: actorId,
		Type:    party2.CommandPartyCreate,
		Body:    party2.CreateCommandBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

func LeaveCommandProvider(actorId uint32, partyId uint32, force bool) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(actorId))
	value := &party2.Command[party2.LeaveCommandBody]{
		ActorId: actorId,
		Type:    party2.CommandPartyLeave,
		Body: party2.LeaveCommandBody{
			PartyId: partyId,
			Force:   force,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func ChangeLeaderCommandProvider(actorId uint32, partyId uint32, leaderId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(actorId))
	value := &party2.Command[party2.ChangeLeaderBody]{
		ActorId: actorId,
		Type:    party2.CommandPartyChangeLeader,
		Body: party2.ChangeLeaderBody{
			LeaderId: leaderId,
			PartyId:  partyId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func RequestInviteCommandProvider(actorId uint32, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(actorId))
	value := &party2.Command[party2.RequestInviteBody]{
		ActorId: actorId,
		Type:    party2.CommandPartyRequestInvite,
		Body: party2.RequestInviteBody{
			CharacterId: characterId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
