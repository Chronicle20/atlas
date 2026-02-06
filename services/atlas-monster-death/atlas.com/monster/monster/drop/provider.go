package drop

import (
	"context"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

func byMonsterIdProvider(l logrus.FieldLogger) func(ctx context.Context) func(monsterId uint32) model.Provider[[]Model] {
	return func(ctx context.Context) func(monsterId uint32) model.Provider[[]Model] {
		return func(monsterId uint32) model.Provider[[]Model] {
			return requests.SliceProvider[RestModel, Model](l, ctx)(requestForMonster(monsterId), Extract, model.Filters[Model]())
		}
	}
}

func GetByMonsterId(l logrus.FieldLogger) func(ctx context.Context) func(monsterId uint32) ([]Model, error) {
	return func(ctx context.Context) func(monsterId uint32) ([]Model, error) {
		return func(monsterId uint32) ([]Model, error) {
			return byMonsterIdProvider(l)(ctx)(monsterId)()
		}
	}
}
