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

func getBySlot(tenantId uuid.UUID, compartmentId uuid.UUID, slot int16) database.EntityProvider[Entity] {
	return func(db *gorm.DB) model.Provider[Entity] {
		return database.Query[Entity](db, &Entity{TenantId: tenantId, CompartmentId: compartmentId, Slot: slot})
	}
}

func getByReferenceId(tenantId uuid.UUID, referenceId uint32, referenceType ReferenceType) database.EntityProvider[Entity] {
	return func(db *gorm.DB) model.Provider[Entity] {
		return database.Query[Entity](db, &Entity{TenantId: tenantId, ReferenceId: referenceId, ReferenceType: string(referenceType)})
	}
}

func getById(tenantId uuid.UUID, id uint32) database.EntityProvider[Entity] {
	return func(db *gorm.DB) model.Provider[Entity] {
		return database.Query[Entity](db, &Entity{TenantId: tenantId, Id: id})
	}
}
