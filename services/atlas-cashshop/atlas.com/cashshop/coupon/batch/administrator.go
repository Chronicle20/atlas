package batch

import (
	"gorm.io/gorm"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// createEntity inserts the batch row that the generated coupons will point at.
// It is written before the coupons, inside the same transaction, so a batch row
// can never outlive a failed generation.
func createEntity(db *gorm.DB, t tenant.Model, m Model) (Model, error) {
	e := &Entity{
		Id:             m.Id(),
		TenantId:       t.Id(),
		Description:    m.Description(),
		RequestedCount: m.RequestedCount(),
		GeneratedCount: m.GeneratedCount(),
	}

	if err := db.Create(e).Error; err != nil {
		return Model{}, err
	}
	return Make(*e)
}
