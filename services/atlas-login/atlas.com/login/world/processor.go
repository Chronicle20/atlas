package world

import (
	"atlas-login/channel"
	"atlas-login/tenant"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

func AllProvider(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) model.SliceProvider[Model] {
	return requests.SliceProvider[RestModel, Model](l)(requestWorlds(l, span, tenant), Extract)
}

func GetAll(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model, decorators ...model.Decorator[Model]) ([]Model, error) {
	return model.SliceMap(AllProvider(l, span, tenant), model.Decorate(decorators...))()
}

func ByIdModelProvider(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(worldId byte) model.Provider[Model] {
	return func(worldId byte) model.Provider[Model] {
		return requests.Provider[RestModel, Model](l)(requestWorld(l, span, tenant)(worldId), Extract)
	}
}

func GetById(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(worldId byte) (Model, error) {
	return func(worldId byte) (Model, error) {
		return ByIdModelProvider(l, span, tenant)(worldId)()
	}
}

func GetCapacityStatus(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(worldId byte) Status {
	return func(worldId byte) Status {
		w, err := GetById(l, span, tenant)(worldId)
		if err != nil {
			return StatusFull
		}
		return w.CapacityStatus()
	}
}

func ChannelLoadDecorator(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) model.Decorator[Model] {
	return func(m Model) Model {
		nm, err := model.Fold[channel.Model, Model](channel.ByWorldModelProvider(l, span, tenant)(m.Id()), Clone(m), foldChannelLoad)()
		if err != nil {
			return m
		}
		return nm
	}
}

func foldChannelLoad(m Model, c channel.Model) (Model, error) {
	return CloneWorld(m).AddChannelLoad(c.Id(), c.Capacity()).Build(), nil
}
