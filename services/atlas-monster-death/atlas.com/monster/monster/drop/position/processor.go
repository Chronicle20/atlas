package position

import (
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

func GetInMap(l logrus.FieldLogger) func(ctx context.Context) func(mapId uint32, initialX int16, initialY int16, fallbackX int16, fallbackY int16) model.Provider[Model] {
	return func(ctx context.Context) func(mapId uint32, initialX int16, initialY int16, fallbackX int16, fallbackY int16) model.Provider[Model] {
		return func(mapId uint32, initialX int16, initialY int16, fallbackX int16, fallbackY int16) model.Provider[Model] {
			return requests.Provider[RestModel, Model](l, ctx)(getInMap(mapId, initialX, initialY, fallbackX, fallbackY), Extract)
		}
	}
}
