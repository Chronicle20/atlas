package portal

import (
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

func inMapByIdModelProvider(l logrus.FieldLogger, ctx context.Context) func(mapId uint32, id uint32) model.Provider[Model] {
	return func(mapId uint32, id uint32) model.Provider[Model] {
		return requests.Provider[RestModel, Model](l, ctx)(requestInMapById(mapId, id), Extract)
	}
}

func GetInMapById(l logrus.FieldLogger, ctx context.Context) func(mapId uint32, id uint32) (Model, error) {
	return func(mapId uint32, id uint32) (Model, error) {
		return inMapByIdModelProvider(l, ctx)(mapId, id)()
	}
}
