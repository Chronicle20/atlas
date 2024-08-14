package monster

import (
	"atlas-channel/tenant"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

func GetById(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(uniqueId uint32) (Model, error) {
	return func(uniqueId uint32) (Model, error) {
		return requests.Provider[RestModel, Model](l)(requestById(l, span, tenant)(uniqueId), Extract)()
	}
}

func InMapModelProvider(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(worldId byte, channelId byte, mapId uint32) model.Provider[[]Model] {
	return func(worldId byte, channelId byte, mapId uint32) model.Provider[[]Model] {
		return requests.SliceProvider[RestModel, Model](l)(requestInMap(l, span, tenant)(worldId, channelId, mapId), Extract)
	}
}

func ForEachInMap(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(worldId byte, channelId byte, mapId uint32, f model.Operator[Model]) error {
	return func(worldId byte, channelId byte, mapId uint32, f model.Operator[Model]) error {
		return model.ForEachSlice(InMapModelProvider(l, span, tenant)(worldId, channelId, mapId), f, model.ParallelExecute())
	}
}

func GetInMap(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(worldId byte, channelId byte, mapId uint32) ([]Model, error) {
	return func(worldId byte, channelId byte, mapId uint32) ([]Model, error) {
		return InMapModelProvider(l, span, tenant)(worldId, channelId, mapId)()
	}
}
