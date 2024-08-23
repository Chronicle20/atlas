package information

import (
	"atlas-monsters/tenant"
	"context"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

func GetById(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(monsterId uint32) (Model, error) {
	return func(monsterId uint32) (Model, error) {
		return requests.Provider[RestModel, Model](l)(requestById(ctx, tenant)(monsterId), Extract)()
	}
}
