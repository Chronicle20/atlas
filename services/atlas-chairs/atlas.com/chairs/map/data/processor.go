package data

import (
	"context"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

func GetById(l logrus.FieldLogger) func(ctx context.Context) func(mapId uint32) (Model, error) {
	return func(ctx context.Context) func(mapId uint32) (Model, error) {
		return func(mapId uint32) (Model, error) {
			return requests.Provider[RestModel, Model](l, ctx)(requestMap(mapId), Extract)()
		}
	}
}
