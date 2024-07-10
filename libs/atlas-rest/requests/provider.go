package requests

import (
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
)

//goland:noinspection GoUnusedExportedFunction
func Provider[A any, M any](l logrus.FieldLogger) func(r Request[A], t model.Transformer[A, M]) model.Provider[M] {
	return func(r Request[A], t model.Transformer[A, M]) model.Provider[M] {
		result, err := r(l)
		if err != nil {
			return model.ErrorProvider[M](err)
		}
		return model.Map[A, M](model.FixedProvider(result), t)
	}
}

//goland:noinspection GoUnusedExportedFunction
func SliceProvider[A any, M any](l logrus.FieldLogger) func(r Request[[]A], t model.Transformer[A, M], filters ...model.Filter[M]) model.SliceProvider[M] {
	return func(r Request[[]A], t model.Transformer[A, M], filters ...model.Filter[M]) model.SliceProvider[M] {
		resp, err := r(l)
		if err != nil {
			return model.ErrorSliceProvider[M](err)
		}
		sm := model.SliceMap[A, M](model.FixedSliceProvider(resp), t)
		return model.FilteredProvider[M](sm, filters...)
	}
}
