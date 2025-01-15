package key

import (
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

func ByCharacterIdProvider(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32) model.Provider[[]Model] {
	return func(ctx context.Context) func(characterId uint32) model.Provider[[]Model] {
		return func(characterId uint32) model.Provider[[]Model] {
			return requests.SliceProvider[RestModel, Model](l, ctx)(requestByCharacterId(characterId), Extract, model.Filters[Model]())
		}
	}
}

func Extract(rm RestModel) (Model, error) {
	return Model{
		key:     rm.Key,
		theType: rm.Type,
		action:  rm.Action,
	}, nil
}

func GetByCharacterId(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32) ([]Model, error) {
	return func(ctx context.Context) func(characterId uint32) ([]Model, error) {
		return func(characterId uint32) ([]Model, error) {
			return ByCharacterIdProvider(l)(ctx)(characterId)()
		}
	}
}
