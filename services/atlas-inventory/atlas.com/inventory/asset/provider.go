package asset

import (
	"atlas-inventory/database"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func getByCompartmentId(tenantId uuid.UUID, compartmentId uuid.UUID) database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		return database.SliceQuery[Entity](db, &Entity{TenantId: tenantId, CompartmentId: compartmentId})
	}
}
