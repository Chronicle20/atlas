package seed

import (
	"atlas-character-factory/kafka/message/seed"

	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
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
