package portal

import (
	"atlas-character/tenant"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

func inMapByIdModelProvider(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(mapId uint32, id uint32) model.Provider[Model] {
	return func(mapId uint32, id uint32) model.Provider[Model] {
		return requests.Provider[RestModel, Model](l)(requestInMapById(ctx, tenant)(mapId, id), Extract)
	}
}

func GetInMapById(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(mapId uint32, id uint32) (Model, error) {
	return func(mapId uint32, id uint32) (Model, error) {
		return inMapByIdModelProvider(l, ctx, tenant)(mapId, id)()
	}
}
