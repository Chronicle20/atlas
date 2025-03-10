package pet

import (
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

func ByOwnerProvider(l logrus.FieldLogger) func(ctx context.Context) func(ownerId uint32) model.Provider[[]Model] {
	return func(ctx context.Context) func(ownerId uint32) model.Provider[[]Model] {
		return func(ownerId uint32) model.Provider[[]Model] {
			return requests.SliceProvider[RestModel, Model](l, ctx)(requestByOwnerId(ownerId), Extract, model.Filters[Model]())
		}
	}
}

func GetByOwner(l logrus.FieldLogger) func(ctx context.Context) func(ownerId uint32) ([]Model, error) {
	return func(ctx context.Context) func(ownerId uint32) ([]Model, error) {
		return func(ownerId uint32) ([]Model, error) {
			return ByOwnerProvider(l)(ctx)(ownerId)()
		}
	}
}

func GetByOwnerItem(l logrus.FieldLogger) func(ctx context.Context) func(ownerId uint32, inventoryItemId uint32) (Model, error) {
	return func(ctx context.Context) func(ownerId uint32, inventoryItemId uint32) (Model, error) {
		return func(ownerId uint32, inventoryItemId uint32) (Model, error) {
			return model.FirstProvider(ByOwnerProvider(l)(ctx)(ownerId), model.Filters(InventoryItemIdFilter(inventoryItemId)))()
		}
	}
}

func InventoryItemIdFilter(inventoryItemId uint32) model.Filter[Model] {
	return func(m Model) bool {
		return m.InventoryItemId() == inventoryItemId
	}
}
