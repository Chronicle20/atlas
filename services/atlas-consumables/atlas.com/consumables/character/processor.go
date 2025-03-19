package character

import (
	character2 "atlas-consumables/kafka/message/character"
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

func GetByIdWithInventory(l logrus.FieldLogger) func(ctx context.Context) func(decorators ...model.Decorator[Model]) func(characterId uint32) (Model, error) {
	return func(ctx context.Context) func(decorators ...model.Decorator[Model]) func(characterId uint32) (Model, error) {
		return func(decorators ...model.Decorator[Model]) func(characterId uint32) (Model, error) {
			return func(characterId uint32) (Model, error) {
				p := requests.Provider[RestModel, Model](l, ctx)(requestByIdWithInventory(characterId), Extract)
				return model.Map(model.Decorate(decorators))(p)()
			}
		}
	}
}

func ChangeMap(l logrus.FieldLogger) func(ctx context.Context) func(m _map.Model, characterId uint32, portalId uint32) error {
	return func(ctx context.Context) func(m _map.Model, characterId uint32, portalId uint32) error {
		return func(m _map.Model, characterId uint32, portalId uint32) error {
			return producer.ProviderImpl(l)(ctx)(character2.EnvCommandTopic)(changeMapProvider(m, characterId, portalId))
		}
	}
}

func ChangeHP(l logrus.FieldLogger) func(ctx context.Context) func(m _map.Model, characterId uint32, amount int16) error {
	return func(ctx context.Context) func(m _map.Model, characterId uint32, amount int16) error {
		return func(m _map.Model, characterId uint32, amount int16) error {
			return producer.ProviderImpl(l)(ctx)(character2.EnvCommandTopic)(changeHPCommandProvider(m, characterId, amount))
		}
	}
}

func ChangeMP(l logrus.FieldLogger) func(ctx context.Context) func(m _map.Model, characterId uint32, amount int16) error {
	return func(ctx context.Context) func(m _map.Model, characterId uint32, amount int16) error {
		return func(m _map.Model, characterId uint32, amount int16) error {
			return producer.ProviderImpl(l)(ctx)(character2.EnvCommandTopic)(changeMPCommandProvider(m, characterId, amount))
		}
	}
}
