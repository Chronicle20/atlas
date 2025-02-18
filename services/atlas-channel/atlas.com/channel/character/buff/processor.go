package buff

import (
	"atlas-channel/kafka/producer"
	"atlas-channel/skill/effect/statup"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

func byCharacterIdProvider(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32) model.Provider[[]Model] {
	return func(ctx context.Context) func(characterId uint32) model.Provider[[]Model] {
		return func(characterId uint32) model.Provider[[]Model] {
			return requests.SliceProvider[RestModel, Model](l, ctx)(requestById(characterId), Extract, model.Filters[Model]())
		}
	}
}

func GetByCharacterId(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32) ([]Model, error) {
	return func(ctx context.Context) func(characterId uint32) ([]Model, error) {
		return func(characterId uint32) ([]Model, error) {
			return byCharacterIdProvider(l)(ctx)(characterId)()
		}
	}
}

func Apply(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, characterId uint32, sourceId uint32, duration int32, statups []statup.Model) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, characterId uint32, sourceId uint32, duration int32, statups []statup.Model) error {
		return func(worldId byte, channelId byte, characterId uint32, sourceId uint32, duration int32, statups []statup.Model) error {
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(applyCommandProvider(worldId, channelId, characterId, sourceId, duration, statups))
		}
	}
}
