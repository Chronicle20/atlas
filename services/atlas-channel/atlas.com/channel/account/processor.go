package account

import (
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

type LoginErr string

func ByIdModelProvider(l logrus.FieldLogger) func(ctx context.Context) func(id uint32) model.Provider[Model] {
	return func(ctx context.Context) func(id uint32) model.Provider[Model] {
		return func(id uint32) model.Provider[Model] {
			return requests.Provider[RestModel, Model](l, ctx)(requestAccountById(id), Extract)
		}
	}
}

func AllProvider(l logrus.FieldLogger) func(ctx context.Context) model.Provider[[]Model] {
	return func(ctx context.Context) model.Provider[[]Model] {
		return requests.SliceProvider[RestModel, Model](l, ctx)(requestAccounts, Extract, model.Filters[Model]())
	}
}

func GetById(l logrus.FieldLogger) func(ctx context.Context) func(id uint32) (Model, error) {
	return func(ctx context.Context) func(id uint32) (Model, error) {
		return func(id uint32) (Model, error) {
			return ByIdModelProvider(l)(ctx)(id)()
		}
	}
}

func GetAll(l logrus.FieldLogger) func(ctx context.Context) ([]Model, error) {
	return func(ctx context.Context) ([]Model, error) {
		return AllProvider(l)(ctx)()
	}
}

func IsLoggedIn(ctx context.Context) func(id uint32) bool {
	return func(id uint32) bool {
		return getRegistry().LoggedIn(Key{Tenant: tenant.MustFromContext(ctx), Id: id})
	}
}

func InitializeRegistry(l logrus.FieldLogger) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		as, err := model.CollectToMap[Model, Key, bool](AllProvider(l)(ctx), KeyForTenantFunc(tenant.MustFromContext(ctx)), IsLogged)()
		if err != nil {
			return err
		}
		getRegistry().Init(as)
		return nil
	}
}

func IsLogged(m Model) bool {
	return m.LoggedIn() > 0
}
