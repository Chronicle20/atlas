package character

import (
	"context"

	"atlas-map-actions/kafka/producer"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	kfkProducer "github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

type statusEvent[E any] struct {
	CharacterId uint32   `json:"characterId"`
	Type        string   `json:"type"`
	WorldId     world.Id `json:"worldId"`
	Body        E        `json:"body"`
}

type statChangedBody struct {
	ChannelId       channel.Id `json:"channelId"`
	ExclRequestSent bool       `json:"exclRequestSent"`
}

func EnableActionsProvider(ch channel.Model, characterId uint32) model.Provider[[]kafka.Message] {
	key := kfkProducer.CreateKey(int(characterId))
	value := &statusEvent[statChangedBody]{
		CharacterId: characterId,
		Type:        EventCharacterStatusTypeStatChanged,
		WorldId:     ch.WorldId(),
		Body: statChangedBody{
			ChannelId:       ch.Id(),
			ExclRequestSent: true,
		},
	}
	return kfkProducer.SingleMessageProvider(key, value)
}

func EnableActions(l logrus.FieldLogger) func(ctx context.Context) func(ch channel.Model, characterId uint32) {
	return func(ctx context.Context) func(ch channel.Model, characterId uint32) {
		return func(ch channel.Model, characterId uint32) {
			_ = producer.ProviderImpl(l)(ctx)(EnvEventTopicCharacterStatus)(EnableActionsProvider(ch, characterId))
		}
	}
}
