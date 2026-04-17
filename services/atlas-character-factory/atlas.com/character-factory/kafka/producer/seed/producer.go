package seed

import (
	"atlas-character-factory/kafka/message/seed"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func CreatedEventStatusProvider(accountId uint32, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(accountId))
	value := &seed.StatusEvent[seed.CreatedStatusEventBody]{
		AccountId: accountId,
		Type:      seed.StatusEventTypeCreated,
		Body: seed.CreatedStatusEventBody{
			CharacterId: characterId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// FailedEventStatusProvider emits a FAILED event on EVENT_TOPIC_SEED_STATUS,
// mirroring CreatedEventStatusProvider. Used by the factory saga-status bridge
// to re-emit CharacterCreation failures toward atlas-login.
func FailedEventStatusProvider(accountId uint32, reason string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(accountId))
	value := &seed.StatusEvent[seed.FailedStatusEventBody]{
		AccountId: accountId,
		Type:      seed.StatusEventTypeFailed,
		Body: seed.FailedStatusEventBody{
			Reason: reason,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
