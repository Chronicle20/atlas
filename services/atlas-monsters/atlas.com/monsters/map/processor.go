package _map

import (
	"context"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

func CharacterIdsInFieldProvider(l logrus.FieldLogger) func(ctx context.Context) func(f field.Model) model.Provider[[]uint32] {
	return func(ctx context.Context) func(f field.Model) model.Provider[[]uint32] {
		return func(f field.Model) model.Provider[[]uint32] {
			return requests.SliceProvider[RestModel, uint32](l, ctx)(requestCharactersInField(f), Extract, model.Filters[uint32]())
		}
	}
}

func GetCharacterIdsInField(l logrus.FieldLogger) func(ctx context.Context) func(f field.Model) ([]uint32, error) {
	return func(ctx context.Context) func(f field.Model) ([]uint32, error) {
		return func(f field.Model) ([]uint32, error) {
			return CharacterIdsInFieldProvider(l)(ctx)(f)()
		}
	}
}
