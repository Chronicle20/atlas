package data

import (
	"context"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

func GetById(l logrus.FieldLogger) func(ctx context.Context) func(mapId _map.Id) (Model, error) {
	return func(ctx context.Context) func(mapId _map.Id) (Model, error) {
		return func(mapId _map.Id) (Model, error) {
			return requests.Provider[RestModel, Model](l, ctx)(requestMap(mapId), Extract)()
		}
	}
}
