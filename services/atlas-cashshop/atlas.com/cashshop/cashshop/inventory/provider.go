package inventory

import (
	"atlas-cashshop/cashshop/inventory/asset"
	"atlas-cashshop/cashshop/inventory/compartment"
	"atlas-cashshop/item"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ByAccountIdProvider retrieves the inventory for an account
func ByAccountIdProvider(tenantId uuid.UUID) func(accountId uint32) func(itemProvider func(itemId uint32) model.Provider[item.Model]) func(db *gorm.DB) model.Provider[Model] {
	return func(accountId uint32) func(itemProvider func(itemId uint32) model.Provider[item.Model]) func(db *gorm.DB) model.Provider[Model] {
		return func(itemProvider func(itemId uint32) model.Provider[item.Model]) func(db *gorm.DB) model.Provider[Model] {
			return func(db *gorm.DB) model.Provider[Model] {
				return func() (Model, error) {
					// Get all compartments for the account
					compartmentsProvider := compartment.AllByAccountIdProvider(tenantId)(accountId)(db)
					compartments, err := compartmentsProvider()
					if err != nil {
						return Model{}, err
					}

					// Create the inventory builder
					builder := NewBuilder(accountId)

					// For each compartment, get its assets and add them to the compartment
					for _, c := range compartments {
						// Get all assets for this compartment
						assetsProvider := asset.ByCompartmentIdProvider(tenantId)(c.Id())(itemProvider)(db)
						assets, err := assetsProvider()
						if err != nil {
							return Model{}, err
						}

						// Create a new compartment builder with the assets
						compartmentBuilder := compartment.Clone(c)
						compartmentBuilder.SetAssets(assets)

						// Add the compartment to the inventory
						builder.SetCompartment(compartmentBuilder.Build())
					}

					// Build and return the inventory
					return builder.Build(), nil
				}
			}
		}
	}
}
