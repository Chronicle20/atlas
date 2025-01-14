package party

import (
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func createCommandProvider(leaderId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(leaderId))
	value := &commandEvent[createCommandBody]{
		Type: CommandPartyCreate,
		Body: createCommandBody{
			LeaderId: leaderId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func joinCommandProvider(partyId uint32, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &commandEvent[joinCommandBody]{
		Type: CommandPartyJoin,
		Body: joinCommandBody{
			CharacterId: characterId,
			PartyId:     partyId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func leaveCommandProvider(partyId uint32, characterId uint32, force bool) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &commandEvent[leaveCommandBody]{
		Type: CommandPartyLeave,
		Body: leaveCommandBody{
			CharacterId: characterId,
			PartyId:     partyId,
			Force:       force,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func changeLeaderCommandProvider(partyId uint32, leaderId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(leaderId))
	value := &commandEvent[changeLeaderBody]{
		Type: CommandPartyChangeLeader,
		Body: changeLeaderBody{
			LeaderId: leaderId,
			PartyId:  partyId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func createdEventProvider(partyId uint32, worldId byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(partyId))
	value := &statusEvent[createdEventBody]{
		PartyId: partyId,
		WorldId: worldId,
		Type:    EventPartyStatusTypeCreated,
		Body:    createdEventBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

func joinedEventProvider(partyId uint32, worldId byte, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(partyId))
	value := &statusEvent[joinedEventBody]{
		PartyId: partyId,
		WorldId: worldId,
		Type:    EventPartyStatusTypeJoined,
		Body: joinedEventBody{
			CharacterId: characterId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func leftEventProvider(partyId uint32, worldId byte, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(partyId))
	value := &statusEvent[leftEventBody]{
		PartyId: partyId,
		WorldId: worldId,
		Type:    EventPartyStatusTypeLeft,
		Body: leftEventBody{
			CharacterId: characterId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func disbandEventProvider(partyId uint32, worldId byte, characterId uint32, members []uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(partyId))
	value := &statusEvent[disbandEventBody]{
		PartyId: partyId,
		WorldId: worldId,
		Type:    EventPartyStatusTypeDisband,
		Body: disbandEventBody{
			CharacterId: characterId,
			Members:     members,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
