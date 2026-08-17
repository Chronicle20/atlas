package portal

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// GetByMapId fetches every portal in a map. atlas-data's GET
// /data/maps/{id}/portals is now paginated (task-117), so this drains
// every page rather than fetching one.
func GetByMapId(l logrus.FieldLogger, ctx context.Context) func(mapId uint32) model.Provider[[]Model] {
	return func(mapId uint32) model.Provider[[]Model] {
		url, err := inMapUrl(ctx, mapId)
		if err != nil {
			return model.ErrorProvider[[]Model](err)
		}
		return requests.DrainProvider[RestModel, Model](l, ctx)(url, 250, Extract, model.Filters[Model]())
	}
}
