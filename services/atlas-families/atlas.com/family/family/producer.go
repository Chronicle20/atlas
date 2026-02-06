package family

import (
	"time"

	"atlas-family/kafka/message/family"

	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

// LinkCreatedEventProvider creates a Kafka message provider for link created events
func LinkCreatedEventProvider(worldId world.Id, characterId uint32, seniorId uint32, juniorId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := family.NewLinkCreatedEvent(worldId, characterId, seniorId, juniorId)
	return producer.SingleMessageProvider(key, value)
}

// LinkBrokenEventProvider creates a Kafka message provider for link broken events
func LinkBrokenEventProvider(worldId world.Id, characterId uint32, seniorId uint32, juniorId uint32, reason string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := family.NewLinkBrokenEvent(worldId, characterId, seniorId, juniorId, reason)
	return producer.SingleMessageProvider(key, value)
}

// RepGainedEventProvider creates a Kafka message provider for reputation gained events
func RepGainedEventProvider(worldId world.Id, characterId uint32, repGained uint32, dailyRep uint32, source string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := family.NewRepGainedEvent(worldId, characterId, repGained, dailyRep, source)
	return producer.SingleMessageProvider(key, value)
}

// RepRedeemedEventProvider creates a Kafka message provider for reputation redeemed events
func RepRedeemedEventProvider(worldId world.Id, characterId uint32, repRedeemed uint32, reason string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := family.NewRepRedeemedEvent(worldId, characterId, repRedeemed, reason)
	return producer.SingleMessageProvider(key, value)
}

// RepErrorEventProvider creates a Kafka message provider for reputation error events
func RepErrorEventProvider(worldId world.Id, characterId uint32, errorCode string, errorMessage string, amount uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := family.NewRepErrorEvent(worldId, characterId, errorCode, errorMessage, amount)
	return producer.SingleMessageProvider(key, value)
}

// LinkErrorEventProvider creates a Kafka message provider for link error events
func LinkErrorEventProvider(worldId world.Id, characterId uint32, seniorId uint32, juniorId uint32, errorCode string, errorMessage string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := family.NewLinkErrorEvent(worldId, characterId, seniorId, juniorId, errorCode, errorMessage)
	return producer.SingleMessageProvider(key, value)
}

// TreeDissolvedEventProvider creates a Kafka message provider for tree dissolved events

// RepResetEventProvider creates a Kafka message provider for reputation reset events
func RepResetEventProvider(worldId world.Id, characterId uint32, previousDailyRep uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &family.Event[family.RepResetEventBody]{
		WorldId:     worldId,
		CharacterId: characterId,
		Type:        family.EventTypeRepReset,
		Body: family.RepResetEventBody{
			PreviousDailyRep: previousDailyRep,
			Timestamp:        time.Now(),
		},
	}
	return producer.SingleMessageProvider(key, value)
}
