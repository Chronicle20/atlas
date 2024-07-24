package character

import (
	"atlas-login/tenant"
	"errors"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"regexp"
)

func IsValidName(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(name string) (bool, error) {
	return func(name string) (bool, error) {
		m, err := regexp.MatchString("[A-Za-z0-9\u3040-\u309F\u30A0-\u30FF\u4E00-\u9FAF]{3,12}", name)
		if err != nil {
			return false, err
		}
		if !m {
			return false, nil
		}

		cs, err := GetByName(l, span, tenant)(name)
		if len(cs) != 0 || err != nil {
			return false, nil
		}

		//TODO
		//bn, err := blocked_name.IsBlockedName(l, span)(name)
		//if bn {
		//	return false, err
		//}

		return true, nil
	}
}

func byAccountAndWorldProvider(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(accountId uint32, worldId byte) model.Provider[[]Model] {
	return func(accountId uint32, worldId byte) model.Provider[[]Model] {
		return requests.SliceProvider[RestModel, Model](l)(requestByAccountAndWorld(l, span, tenant)(accountId, worldId), Extract)
	}
}

func GetForWorld(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(accountId uint32, worldId byte) ([]Model, error) {
	return func(accountId uint32, worldId byte) ([]Model, error) {
		cs, err := byAccountAndWorldProvider(l, span, tenant)(accountId, worldId)()
		if errors.Is(requests.ErrNotFound, err) {
			return make([]Model, 0), nil
		}
		return cs, err
	}
}

func byNameProvider(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(name string) model.Provider[[]Model] {
	return func(name string) model.Provider[[]Model] {
		return requests.SliceProvider[RestModel, Model](l)(requestByName(l, span, tenant)(name), Extract)
	}
}

func GetByName(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(name string) ([]Model, error) {
	return func(name string) ([]Model, error) {
		return byNameProvider(l, span, tenant)(name)()
	}
}

func GetById(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(characterId uint32) (Model, error) {
	return func(characterId uint32) (Model, error) {
		return requests.Provider[RestModel, Model](l)(requestById(l, span, tenant)(characterId), Extract)()
	}
}

func DeleteById(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(characterId uint32) error {
	return func(characterId uint32) error {
		return requestDelete(l, span, tenant)(characterId)(l)
	}
}
