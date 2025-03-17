package character

import (
	"atlas-consumables/kafka/producer"
	"context"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

func GetById(l logrus.FieldLogger) func(ctx context.Context) func(decorators ...model.Decorator[Model]) func(characterId uint32) (Model, error) {
	return func(ctx context.Context) func(decorators ...model.Decorator[Model]) func(characterId uint32) (Model, error) {
		return func(decorators ...model.Decorator[Model]) func(characterId uint32) (Model, error) {
			return func(characterId uint32) (Model, error) {
				p := requests.Provider[RestModel, Model](l, ctx)(requestById(characterId), Extract)
				return model.Map(model.Decorate(decorators))(p)()
			}
		}
	}
}

func ChangeHP(l logrus.FieldLogger) func(ctx context.Context) func(m _map.Model, characterId uint32, amount int16) error {
	return func(ctx context.Context) func(m _map.Model, characterId uint32, amount int16) error {
		return func(m _map.Model, characterId uint32, amount int16) error {
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(changeHPCommandProvider(m, characterId, amount))
		}
	}
}

func ChangeMP(l logrus.FieldLogger) func(ctx context.Context) func(m _map.Model, characterId uint32, amount int16) error {
	return func(ctx context.Context) func(m _map.Model, characterId uint32, amount int16) error {
		return func(m _map.Model, characterId uint32, amount int16) error {
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(changeMPCommandProvider(m, characterId, amount))
		}
	}
}
