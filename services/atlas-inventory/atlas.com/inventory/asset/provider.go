package asset

import (
	database "github.com/Chronicle20/atlas/libs/atlas-database"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func getByCompartmentId(compartmentId uuid.UUID) database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		return database.SliceQuery[Entity](db, &Entity{CompartmentId: compartmentId})
	}
}

func getByCompartmentIdPaged(compartmentId uuid.UUID, page model.Page) database.EntityProvider[model.Paged[Entity]] {
	return func(db *gorm.DB) model.Provider[model.Paged[Entity]] {
		return database.PagedQuery[Entity](db.Where(&Entity{CompartmentId: compartmentId}), page)
	}
}

func getBySlot(compartmentId uuid.UUID, slot int16) database.EntityProvider[Entity] {
	return func(db *gorm.DB) model.Provider[Entity] {
		return database.Query[Entity](db, &Entity{CompartmentId: compartmentId, Slot: slot})
	}
}

func getById(id uint32) database.EntityProvider[Entity] {
	return func(db *gorm.DB) model.Provider[Entity] {
		return database.Query[Entity](db, &Entity{Id: id})
	}
}
