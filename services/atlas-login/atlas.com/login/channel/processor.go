package channel

import (
	"atlas-login/tenant"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

func ByWorldModelProvider(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(worldId byte) model.Provider[[]Model] {
	return func(worldId byte) model.Provider[[]Model] {
		return requests.SliceProvider[RestModel, Model](l)(requestChannelsForWorld(ctx, tenant)(worldId), Extract)
	}
}
