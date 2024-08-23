package world

import (
	"atlas-login/channel"
	"atlas-login/tenant"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

func AllProvider(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](l)(requestWorlds(ctx, tenant), Extract)
}

func GetAll(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model, decorators ...model.Decorator[Model]) ([]Model, error) {
	return model.SliceMap(AllProvider(l, ctx, tenant), model.Decorate(decorators...))()
}

func ByIdModelProvider(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(worldId byte) model.Provider[Model] {
	return func(worldId byte) model.Provider[Model] {
		return requests.Provider[RestModel, Model](l)(requestWorld(ctx, tenant)(worldId), Extract)
	}
}

func GetById(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(worldId byte) (Model, error) {
	return func(worldId byte) (Model, error) {
		return ByIdModelProvider(l, ctx, tenant)(worldId)()
	}
}

func GetCapacityStatus(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(worldId byte) Status {
	return func(worldId byte) Status {
		w, err := GetById(l, ctx, tenant)(worldId)
		if err != nil {
			return StatusFull
		}
		return w.CapacityStatus()
	}
}

func ChannelLoadDecorator(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) model.Decorator[Model] {
	return func(m Model) Model {
		nm, err := model.Fold[channel.Model, Model](channel.ByWorldModelProvider(l, ctx, tenant)(m.Id()), Clone(m), foldChannelLoad)()
		if err != nil {
			return m
		}
		return nm
	}
}

func foldChannelLoad(m Model, c channel.Model) (Model, error) {
	return CloneWorld(m).AddChannelLoad(c.Id(), c.Capacity()).Build(), nil
}
