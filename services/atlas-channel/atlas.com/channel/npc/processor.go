package npc

import (
	"atlas-channel/tenant"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

func ForEachInMap(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(mapId uint32, f model.Operator[Model]) error {
	return func(mapId uint32, f model.Operator[Model]) error {
		return model.ForEachSlice(InMapModelProvider(l, span, tenant)(mapId), f, model.ParallelExecute())
	}
}

func InMapModelProvider(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(mapId uint32) model.Provider[[]Model] {
	return func(mapId uint32) model.Provider[[]Model] {
		return requests.SliceProvider[RestModel, Model](l)(requestNPCsInMap(l, span, tenant)(mapId), Extract)
	}
}

func InMapByObjectIdModelProvider(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(mapId uint32, objectId uint32) model.Provider[[]Model] {
	return func(mapId uint32, objectId uint32) model.Provider[[]Model] {
		return requests.SliceProvider[RestModel, Model](l)(requestNPCsInMapByObjectId(l, span, tenant)(mapId, objectId), Extract)
	}
}

func GetInMapByObjectId(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(mapId uint32, objectId uint32) (Model, error) {
	return func(mapId uint32, objectId uint32) (Model, error) {
		p := InMapByObjectIdModelProvider(l, span, tenant)(mapId, objectId)
		return model.First[Model](p)
	}
}
