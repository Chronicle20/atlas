package skill

import (
	"atlas-messages/kafka/producer"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
	"time"
)

func byCharacterIdProvider(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32) model.Provider[[]Model] {
	return func(ctx context.Context) func(characterId uint32) model.Provider[[]Model] {
		return func(characterId uint32) model.Provider[[]Model] {
			return requests.SliceProvider[RestModel, Model](l, ctx)(requestByCharacterId(characterId), Extract, model.Filters[Model]())
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

func RequestCreate(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, skillId uint32, level byte, masterLevel byte, expiration time.Time) error {
	return func(ctx context.Context) func(characterId uint32, skillId uint32, level byte, masterLevel byte, expiration time.Time) error {
		return func(characterId uint32, skillId uint32, level byte, masterLevel byte, expiration time.Time) error {
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(createCommandProvider(characterId, skillId, level, masterLevel, expiration))
		}
	}
}

func RequestUpdate(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, skillId uint32, level byte, masterLevel byte, expiration time.Time) error {
	return func(ctx context.Context) func(characterId uint32, skillId uint32, level byte, masterLevel byte, expiration time.Time) error {
		return func(characterId uint32, skillId uint32, level byte, masterLevel byte, expiration time.Time) error {
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(updateCommandProvider(characterId, skillId, level, masterLevel, expiration))
		}
	}
}
