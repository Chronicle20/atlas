package cashshop

import (
	"context"

	"github.com/sirupsen/logrus"
)

// GetCompartments retrieves all cash shop compartments from atlas-cashshop
func GetCompartments(l logrus.FieldLogger) func(ctx context.Context) func(accountId uint32) ([]CompartmentRestModel, error) {
	return func(ctx context.Context) func(accountId uint32) ([]CompartmentRestModel, error) {
		return func(accountId uint32) ([]CompartmentRestModel, error) {
			return requestCompartments(accountId)(l, ctx)
		}
	}
}

// GetAllItems retrieves all items across all compartments for an account
func GetAllItems(l logrus.FieldLogger) func(ctx context.Context) func(accountId uint32) ([]ItemRestModel, error) {
	return func(ctx context.Context) func(accountId uint32) ([]ItemRestModel, error) {
		return func(accountId uint32) ([]ItemRestModel, error) {
			comps, err := GetCompartments(l)(ctx)(accountId)
			if err != nil {
				return nil, err
			}

			var allItems []ItemRestModel
			for _, comp := range comps {
				for _, asset := range comp.Assets {
					allItems = append(allItems, asset.Item)
				}
			}

			return allItems, nil
		}
	}
}
