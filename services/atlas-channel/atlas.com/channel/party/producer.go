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
