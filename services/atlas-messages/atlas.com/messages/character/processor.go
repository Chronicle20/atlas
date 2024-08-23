package character

import (
	"atlas-messages/tenant"
	"context"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

func GetById(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(characterId uint32) (Model, error) {
	return func(characterId uint32) (Model, error) {
		return requests.Provider[RestModel, Model](l)(requestById(ctx, tenant)(characterId), Extract)()
	}
}
