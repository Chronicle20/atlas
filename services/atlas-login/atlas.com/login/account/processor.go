package account

import (
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

type LoginErr string

func ForAccountByName(l logrus.FieldLogger, ctx context.Context) func(name string, operator model.Operator[Model]) {
	return func(name string, operator model.Operator[Model]) {
		_ = model.For[Model](ByNameModelProvider(l, ctx)(name), operator)
	}
}

func ForAccountById(l logrus.FieldLogger, ctx context.Context) func(id uint32, operator model.Operator[Model]) {
	return func(id uint32, operator model.Operator[Model]) {
		_ = model.For[Model](ByIdModelProvider(l, ctx)(id), operator)
	}
}

func ByNameModelProvider(l logrus.FieldLogger, ctx context.Context) func(name string) model.Provider[Model] {
	return func(name string) model.Provider[Model] {
		return requests.Provider[RestModel, Model](l, ctx)(requestAccountByName(name), Extract)
	}
}

func ByIdModelProvider(l logrus.FieldLogger, ctx context.Context) func(id uint32) model.Provider[Model] {
	return func(id uint32) model.Provider[Model] {
		return requests.Provider[RestModel, Model](l, ctx)(requestAccountById(id), Extract)
	}
}

func allProvider(l logrus.FieldLogger, ctx context.Context) model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](l, ctx)(requestAccounts(), Extract, model.Filters[Model]())
}

func GetById(l logrus.FieldLogger, ctx context.Context) func(id uint32) (Model, error) {
	return func(id uint32) (Model, error) {
		return ByIdModelProvider(l, ctx)(id)()
	}
}

func GetByName(l logrus.FieldLogger, ctx context.Context) func(name string) (Model, error) {
	return func(name string) (Model, error) {
		return ByNameModelProvider(l, ctx)(name)()
	}
}

func IsLoggedIn(_ logrus.FieldLogger, tenant tenant.Model) func(id uint32) bool {
	return func(id uint32) bool {
		return getRegistry().LoggedIn(Key{Tenant: tenant, Id: id})
	}
}

func InitializeRegistry(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) error {
	as, err := model.CollectToMap[Model, Key, bool](allProvider(l, ctx), KeyForTenantFunc(tenant), IsLogged)()
	if err != nil {
		return err
	}
	getRegistry().Init(as)
	return nil
}

func IsLogged(m Model) bool {
	return m.LoggedIn() > 0
}

func UpdatePin(l logrus.FieldLogger, ctx context.Context) func(id uint32, pin string) error {
	return func(id uint32, pin string) error {
		a, err := GetById(l, ctx)(id)
		if err != nil {
			return err
		}
		a.pin = pin
		_, err = requestUpdate(a)(l, ctx)
		if err != nil {
			return err
		}
		return nil
	}
}

func UpdatePic(l logrus.FieldLogger, ctx context.Context) func(id uint32, pic string) error {
	return func(id uint32, pic string) error {
		a, err := GetById(l, ctx)(id)
		if err != nil {
			return err
		}
		a.pic = pic
		_, err = requestUpdate(a)(l, ctx)
		if err != nil {
			return err
		}
		return nil
	}
}

func UpdateTos(l logrus.FieldLogger, ctx context.Context) func(id uint32, tos bool) error {
	return func(id uint32, tos bool) error {
		a, err := GetById(l, ctx)(id)
		if err != nil {
			return err
		}
		a.tos = tos
		_, err = requestUpdate(a)(l, ctx)
		if err != nil {
			return err
		}
		return nil
	}
}

func UpdateGender(l logrus.FieldLogger, ctx context.Context) func(id uint32, gender byte) error {
	return func(id uint32, gender byte) error {
		a, err := GetById(l, ctx)(id)
		if err != nil {
			return err
		}
		a.gender = gender
		_, err = requestUpdate(a)(l, ctx)
		if err != nil {
			return err
		}
		return nil
	}
}
