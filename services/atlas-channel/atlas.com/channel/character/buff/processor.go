package buff

import (
	"atlas-channel/kafka/producer"
	"atlas-channel/skill/effect/statup"
	"context"
	_map "github.com/Chronicle20/atlas-constants/map"
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

func Apply(l logrus.FieldLogger) func(ctx context.Context) func(m _map.Model, fromId uint32, sourceId uint32, duration int32, statups []statup.Model) model.Operator[uint32] {
	return func(ctx context.Context) func(m _map.Model, fromId uint32, sourceId uint32, duration int32, statups []statup.Model) model.Operator[uint32] {
		return func(m _map.Model, fromId uint32, sourceId uint32, duration int32, statups []statup.Model) model.Operator[uint32] {
			return func(characterId uint32) error {
				return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(applyCommandProvider(m, characterId, fromId, sourceId, duration, statups))
			}
		}
	}
}

func Cancel(l logrus.FieldLogger) func(ctx context.Context) func(m _map.Model, characterId uint32, sourceId uint32) error {
	return func(ctx context.Context) func(m _map.Model, characterId uint32, sourceId uint32) error {
		return func(m _map.Model, characterId uint32, sourceId uint32) error {
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(cancelCommandProvider(m, characterId, sourceId))
		}
	}
}
