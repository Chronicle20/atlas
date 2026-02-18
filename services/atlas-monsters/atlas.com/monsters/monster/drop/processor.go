package drop

import (
	"context"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

func GetByMonsterId(l logrus.FieldLogger) func(ctx context.Context) func(monsterId uint32) ([]Model, error) {
	return func(ctx context.Context) func(monsterId uint32) ([]Model, error) {
		return func(monsterId uint32) ([]Model, error) {
			return requests.SliceProvider[RestModel, Model](l, ctx)(requestForMonster(monsterId), Extract, model.Filters[Model]())()
		}
	}
}
