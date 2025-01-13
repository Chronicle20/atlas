package party

import (
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func createCommandProvider(leaderId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(leaderId))
	value := &commandEvent[createBody]{
		Type: CommandPartyCreate,
		Body: createBody{
			LeaderId: leaderId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func joinCommandProvider(partyId uint32, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &commandEvent[joinBody]{
		Type: CommandPartyJoin,
		Body: joinBody{
			CharacterId: characterId,
			PartyId:     partyId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func leaveCommandProvider(partyId uint32, characterId uint32, force bool) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &commandEvent[leaveBody]{
		Type: CommandPartyLeave,
		Body: leaveBody{
			CharacterId: characterId,
			PartyId:     partyId,
			Force:       force,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
