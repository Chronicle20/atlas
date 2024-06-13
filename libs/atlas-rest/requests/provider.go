package requests

import (
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
)

type Transformer[A any, M any] func(body A) (M, error)

func Provider[A any, M any](l logrus.FieldLogger) func(r Request[A], t Transformer[A, M]) model.Provider[M] {
	return func(r Request[A], t Transformer[A, M]) model.Provider[M] {
		return func() (M, error) {
			var result M
			resp, err := r(l)
			if err != nil {
				return result, err
			}

			result, err = t(resp)
			if err != nil {
				return result, err
			}
			return result, nil
		}
	}
}

func SliceProvider[A any, M any](l logrus.FieldLogger) func(r Request[A], t Transformer[A, M], filters ...model.Filter[M]) model.SliceProvider[M] {
	return func(r Request[A], t Transformer[A, M], filters ...model.Filter[M]) model.SliceProvider[M] {
		return func() ([]M, error) {
			results := make([]M, 0)
			//resp, err := r(l, span)
			//if err != nil {
			//	return results, err
			//}

			//for _, v := range resp.DataList() {
			//	m, err := t(v)
			//	if err != nil {
			//		return nil, err
			//	}
			//	ok := true
			//	for _, filter := range filters {
			//		if !filter(m) {
			//			ok = false
			//			break
			//		}
			//	}
			//	if ok {
			//		results = append(results, m)
			//	}
			//}
			return results, nil
		}
	}
}
