package reactor

import (
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

func InMapProvider(l logrus.FieldLogger) func(ctx context.Context) func(mapId uint32) model.Provider[[]Model] {
	return func(ctx context.Context) func(mapId uint32) model.Provider[[]Model] {
		return func(mapId uint32) model.Provider[[]Model] {
			return requests.SliceProvider[RestModel, Model](l, ctx)(requestReactors(mapId), Extract, model.Filters[Model]())
		}
	}
}
