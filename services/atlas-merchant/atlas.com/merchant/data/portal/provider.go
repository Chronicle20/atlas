package portal

import (
	"context"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

func GetByMapId(l logrus.FieldLogger, ctx context.Context) func(mapId uint32) model.Provider[[]Model] {
	return func(mapId uint32) model.Provider[[]Model] {
		return requests.SliceProvider[RestModel, Model](l, ctx)(requestInMap(mapId), Extract, model.Filters[Model]())
	}
}
