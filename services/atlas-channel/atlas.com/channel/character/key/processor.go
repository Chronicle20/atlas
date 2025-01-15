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

func Update(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, key int32, theType int8, action int32) error {
	return func(ctx context.Context) func(characterId uint32, key int32, theType int8, action int32) error {
		return func(characterId uint32, key int32, theType int8, action int32) error {
			_, err := updateKey(characterId, key, theType, action)(l, ctx)
			return err
		}
	}
}
