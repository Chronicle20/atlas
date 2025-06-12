package inventory

import (
	"atlas-cashshop/cashshop/inventory/asset"
	"atlas-cashshop/cashshop/inventory/compartment"
	"context"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// createDefaultCompartments creates the default compartments for an account
func createDefaultCompartments(db *gorm.DB, t tenant.Model, accountId uint32) ([]compartment.Model, error) {
	// Create Explorer compartment
	explorerEntity := compartment.Entity{
		TenantId:  t.Id(),
		AccountId: accountId,
		Type:      string(compartment.TypeExplorer),
		Capacity:  55,
	}
	if err := db.Create(&explorerEntity).Error; err != nil {
		return nil, err
	}

	// Create Cygnus compartment
	cygnusEntity := compartment.Entity{
		TenantId:  t.Id(),
		AccountId: accountId,
		Type:      string(compartment.TypeCygnus),
		Capacity:  55,
	}
	if err := db.Create(&cygnusEntity).Error; err != nil {
		return nil, err
	}

	// Create Legend compartment
	legendEntity := compartment.Entity{
		TenantId:  t.Id(),
		AccountId: accountId,
		Type:      string(compartment.TypeLegend),
		Capacity:  55,
	}
	if err := db.Create(&legendEntity).Error; err != nil {
		return nil, err
	}

	// Convert entities to models
	explorerModel, err := compartment.Make(explorerEntity)
	if err != nil {
		return nil, err
	}

	cygnusModel, err := compartment.Make(cygnusEntity)
	if err != nil {
		return nil, err
	}

	legendModel, err := compartment.Make(legendEntity)
	if err != nil {
		return nil, err
	}

	return []compartment.Model{explorerModel, cygnusModel, legendModel}, nil
}

// deleteCompartmentsAndAssets deletes all compartments and assets for an account
func deleteCompartmentsAndAssets(ctx context.Context) func(db *gorm.DB, tenantId uuid.UUID, accountId uint32) error {
	return func(db *gorm.DB, tenantId uuid.UUID, accountId uint32) error {
		// Get all compartments for the account
		var compartments []compartment.Entity
		if err := db.WithContext(ctx).Where("tenant_id = ? AND account_id = ?", tenantId, accountId).Find(&compartments).Error; err != nil {
			return err
		}

		// Delete all assets for each compartment
		for _, c := range compartments {
			if err := db.WithContext(ctx).Where("tenant_id = ? AND compartment_id = ?", tenantId, c.Id).Delete(&asset.Entity{}).Error; err != nil {
				return err
			}
		}

		// Delete all compartments
		return db.WithContext(ctx).Where("tenant_id = ? AND account_id = ?", tenantId, accountId).Delete(&compartment.Entity{}).Error
	}
}
