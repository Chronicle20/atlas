package data

import (
	"atlas-character-factory/tenant"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

func byIdModelProvider(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(id uint32) model.Provider[[]Model] {
	return func(id uint32) model.Provider[[]Model] {
		return requests.SliceProvider[RestModel, Model](l)(requestById(ctx, tenant)(id), Extract)
	}
}

func GetById(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(id uint32) ([]Model, error) {
	return func(id uint32) ([]Model, error) {
		return byIdModelProvider(l, ctx, tenant)(id)()
	}
}
