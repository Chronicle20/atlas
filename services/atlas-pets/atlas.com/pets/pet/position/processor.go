package position

import (
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

func GetBelow(l logrus.FieldLogger) func(ctx context.Context) func(mapId uint32, x int16, y int16) model.Provider[Model] {
	return func(ctx context.Context) func(mapId uint32, x int16, y int16) model.Provider[Model] {
		return func(mapId uint32, x int16, y int16) model.Provider[Model] {
			return requests.Provider[FootholdRestModel, Model](l, ctx)(getInMap(mapId, x, y), Extract)
		}
	}
}
