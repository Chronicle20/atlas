package character

import (
	"context"
	"errors"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
	"regexp"
)

func IsValidName(l logrus.FieldLogger, ctx context.Context) func(name string) (bool, error) {
	return func(name string) (bool, error) {
		m, err := regexp.MatchString("[A-Za-z0-9\u3040-\u309F\u30A0-\u30FF\u4E00-\u9FAF]{3,12}", name)
		if err != nil {
			return false, err
		}
		if !m {
			return false, nil
		}

		cs, err := GetByName(l, ctx)(name)
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

func byAccountAndWorldProvider(l logrus.FieldLogger, ctx context.Context) func(accountId uint32, worldId byte) model.Provider[[]Model] {
	return func(accountId uint32, worldId byte) model.Provider[[]Model] {
		return requests.SliceProvider[RestModel, Model](l, ctx)(requestByAccountAndWorld(accountId, worldId), Extract)
	}
}

func GetForWorld(l logrus.FieldLogger, ctx context.Context) func(accountId uint32, worldId byte) ([]Model, error) {
	return func(accountId uint32, worldId byte) ([]Model, error) {
		cs, err := byAccountAndWorldProvider(l, ctx)(accountId, worldId)()
		if errors.Is(requests.ErrNotFound, err) {
			return make([]Model, 0), nil
		}
		return cs, err
	}
}

func byNameProvider(l logrus.FieldLogger, ctx context.Context) func(name string) model.Provider[[]Model] {
	return func(name string) model.Provider[[]Model] {
		return requests.SliceProvider[RestModel, Model](l, ctx)(requestByName(name), Extract)
	}
}

func GetByName(l logrus.FieldLogger, ctx context.Context) func(name string) ([]Model, error) {
	return func(name string) ([]Model, error) {
		return byNameProvider(l, ctx)(name)()
	}
}

func GetById(l logrus.FieldLogger, ctx context.Context) func(characterId uint32) (Model, error) {
	return func(characterId uint32) (Model, error) {
		return requests.Provider[RestModel, Model](l, ctx)(requestById(characterId), Extract)()
	}
}

func DeleteById(l logrus.FieldLogger, ctx context.Context) func(characterId uint32) error {
	return func(characterId uint32) error {
		return requestDelete(characterId)(l, ctx)
	}
}
