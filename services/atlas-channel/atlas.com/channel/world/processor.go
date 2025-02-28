package world

import (
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

func ByIdModelProvider(l logrus.FieldLogger, ctx context.Context) func(worldId byte) model.Provider[Model] {
	return func(worldId byte) model.Provider[Model] {
		return requests.Provider[RestModel, Model](l, ctx)(requestWorld(worldId), Extract)
	}
}

func GetById(l logrus.FieldLogger, ctx context.Context) func(worldId byte) (Model, error) {
	return func(worldId byte) (Model, error) {
		return ByIdModelProvider(l, ctx)(worldId)()
	}
}
