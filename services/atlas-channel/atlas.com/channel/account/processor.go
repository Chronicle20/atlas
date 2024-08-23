package account

import (
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

type LoginErr string

func byIdModelProvider(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(id uint32) model.Provider[Model] {
	return func(id uint32) model.Provider[Model] {
		return requests.Provider[RestModel, Model](l)(requestAccountById(ctx, tenant)(id), Extract)
	}
}

func allProvider(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](l)(requestAccounts(ctx, tenant), Extract)
}

func GetById(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(id uint32) (Model, error) {
	return func(id uint32) (Model, error) {
		return byIdModelProvider(l, ctx, tenant)(id)()
	}
}

func GetAll(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) ([]Model, error) {
	return allProvider(l, ctx, tenant)()
}

func IsLoggedIn(_ logrus.FieldLogger, _ context.Context, tenant tenant.Model) func(id uint32) bool {
	return func(id uint32) bool {
		return getRegistry().LoggedIn(Key{Tenant: tenant, Id: id})
	}
}

func InitializeRegistry(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) error {
	as, err := model.CollectToMap[Model, Key, bool](allProvider(l, ctx, tenant), KeyForTenantFunc(tenant), IsLogged)()
	if err != nil {
		return err
	}
	getRegistry().Init(as)
	return nil
}

func IsLogged(m Model) bool {
	return m.LoggedIn() > 0
}
