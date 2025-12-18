package party

import (
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func createCommandProvider(leaderId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(leaderId))
	value := &commandEvent[createCommandBody]{
		ActorId: leaderId,
		Type:    CommandPartyCreate,
		Body:    createCommandBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

func joinCommandProvider(partyId uint32, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &commandEvent[joinCommandBody]{
		ActorId: characterId,
		Type:    CommandPartyJoin,
		Body: joinCommandBody{
			PartyId: partyId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func leaveCommandProvider(partyId uint32, characterId uint32, force bool) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &commandEvent[leaveCommandBody]{
		ActorId: characterId,
		Type:    CommandPartyLeave,
		Body: leaveCommandBody{
			PartyId: partyId,
			Force:   force,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func changeLeaderCommandProvider(characterId uint32, partyId uint32, leaderId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(leaderId))
	value := &commandEvent[changeLeaderBody]{
		ActorId: characterId,
		Type:    CommandPartyChangeLeader,
		Body: changeLeaderBody{
			LeaderId: leaderId,
			PartyId:  partyId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func createdEventProvider(actorId uint32, partyId uint32, worldId byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(partyId))
	value := &statusEvent[createdEventBody]{
		ActorId: actorId,
		PartyId: partyId,
		WorldId: worldId,
		Type:    EventPartyStatusTypeCreated,
		Body:    createdEventBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

func joinedEventProvider(actorId uint32, partyId uint32, worldId byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(partyId))
	value := &statusEvent[joinedEventBody]{
		ActorId: actorId,
		PartyId: partyId,
		WorldId: worldId,
		Type:    EventPartyStatusTypeJoined,
		Body:    joinedEventBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

func leftEventProvider(actorId uint32, partyId uint32, worldId byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(partyId))
	value := &statusEvent[leftEventBody]{
		ActorId: actorId,
		PartyId: partyId,
		WorldId: worldId,
		Type:    EventPartyStatusTypeLeft,
		Body:    leftEventBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

func expelEventProvider(actorId uint32, partyId uint32, worldId byte, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(partyId))
	value := &statusEvent[expelEventBody]{
		ActorId: actorId,
		PartyId: partyId,
		WorldId: worldId,
		Type:    EventPartyStatusTypeExpel,
		Body: expelEventBody{
			CharacterId: characterId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func disbandEventProvider(actorId uint32, partyId uint32, worldId byte, members []uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(partyId))
	value := &statusEvent[disbandEventBody]{
		ActorId: actorId,
		PartyId: partyId,
		WorldId: worldId,
		Type:    EventPartyStatusTypeDisband,
		Body: disbandEventBody{
			Members: members,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func changeLeaderEventProvider(actorId uint32, partyId uint32, worldId byte, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(partyId))
	value := &statusEvent[changeLeaderEventBody]{
		ActorId: actorId,
		PartyId: partyId,
		WorldId: worldId,
		Type:    EventPartyStatusTypeChangeLeader,
		Body: changeLeaderEventBody{
			CharacterId:  characterId,
			Disconnected: false,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func errorEventProvider(actorId uint32, partyId uint32, worldId byte, errorType string, characterName string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(partyId))
	value := &statusEvent[errorEventBody]{
		ActorId: actorId,
		PartyId: partyId,
		WorldId: worldId,
		Type:    EventPartyStatusTypeError,
		Body: errorEventBody{
			Type:          errorType,
			CharacterName: characterName,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
