package world

import (
	"atlas-login/channel"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

func AllProvider(l logrus.FieldLogger, ctx context.Context) model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](l, ctx)(requestWorlds(), Extract, model.Filters[Model]())
}

func GetAll(l logrus.FieldLogger, ctx context.Context, decorators ...model.Decorator[Model]) ([]Model, error) {
	return model.SliceMap(model.Decorate(decorators))(AllProvider(l, ctx))(model.ParallelMap())()
}

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

func GetCapacityStatus(l logrus.FieldLogger, ctx context.Context) func(worldId byte) Status {
	return func(worldId byte) Status {
		w, err := GetById(l, ctx)(worldId)
		if err != nil {
			return StatusFull
		}
		return w.CapacityStatus()
	}
}

func ChannelLoadDecorator(l logrus.FieldLogger, ctx context.Context) model.Decorator[Model] {
	return func(m Model) Model {
		nm, err := model.Fold[channel.Model, Model](channel.ByWorldModelProvider(l, ctx)(m.Id()), Clone(m), foldChannelLoad)()
		if err != nil {
			return m
		}
		return nm
	}
}

func foldChannelLoad(m Model, c channel.Model) (Model, error) {
	return CloneWorld(m).AddChannelLoad(c.ChannelId(), c.Capacity()).Build(), nil
}
