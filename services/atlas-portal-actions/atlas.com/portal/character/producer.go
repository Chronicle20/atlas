package character

import (
	"context"

	"atlas-portal-actions/kafka/producer"

	kfkProducer "github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// statusEvent represents a character status event
type statusEvent[E any] struct {
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	WorldId     byte   `json:"worldId"`
	Body        E      `json:"body"`
}

// statChangedBody represents the body for a stat changed event
type statChangedBody struct {
	ChannelId       byte `json:"channelId"`
	ExclRequestSent bool `json:"exclRequestSent"`
}

// EnableActionsProvider creates a message provider for enabling character actions
func EnableActionsProvider(worldId byte, channelId byte, characterId uint32) model.Provider[[]kafka.Message] {
	key := kfkProducer.CreateKey(int(characterId))
	value := &statusEvent[statChangedBody]{
		CharacterId: characterId,
		Type:        EventCharacterStatusTypeStatChanged,
		WorldId:     worldId,
		Body: statChangedBody{
			ChannelId:       channelId,
			ExclRequestSent: true,
		},
	}
	return kfkProducer.SingleMessageProvider(key, value)
}

// EnableActions sends an event to enable character actions
func EnableActions(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, characterId uint32) {
	return func(ctx context.Context) func(worldId byte, channelId byte, characterId uint32) {
		return func(worldId byte, channelId byte, characterId uint32) {
			_ = producer.ProviderImpl(l)(ctx)(EnvEventTopicCharacterStatus)(EnableActionsProvider(worldId, channelId, characterId))
		}
	}
}
