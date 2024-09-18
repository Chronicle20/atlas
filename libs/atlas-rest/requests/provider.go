package requests

import (
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
)

//goland:noinspection GoUnusedExportedFunction
func Provider[A any, M any](l logrus.FieldLogger, ctx context.Context) func(r Request[A], t model.Transformer[A, M]) model.Provider[M] {
	return func(r Request[A], t model.Transformer[A, M]) model.Provider[M] {
		result, err := r(l, ctx)
		if err != nil {
			return model.ErrorProvider[M](err)
		}
		return model.Map[A, M](t)(model.FixedProvider(result))
	}
}

//goland:noinspection GoUnusedExportedFunction
func SliceProvider[A any, M any](l logrus.FieldLogger, ctx context.Context) func(r Request[[]A], t model.Transformer[A, M], filters []model.Filter[M]) model.Provider[[]M] {
	return func(r Request[[]A], t model.Transformer[A, M], filters []model.Filter[M]) model.Provider[[]M] {
		resp, err := r(l, ctx)
		if err != nil {
			return model.ErrorProvider[[]M](err)
		}
		sm := model.SliceMap[A, M](t)(model.FixedProvider(resp))()
		return model.FilteredProvider[M](sm, filters)
	}
}
