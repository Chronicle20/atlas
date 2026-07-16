package recipe

import (
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"gorm.io/gorm"
)

// getByItemIdPagedProvider returns a provider that fetches one page of recipe
// entities for the given output itemId, scoped to the active tenant via GORM
// tenant callbacks, ordered by (npc_id, state_id) for deterministic
// responses (PagedQuery appends the primary-key tie-break after this order).
func getByItemIdPagedProvider(itemId uint32, page model.Page) database.EntityProvider[model.Paged[Entity]] {
	return func(db *gorm.DB) model.Provider[model.Paged[Entity]] {
		return database.PagedQuery[Entity](db.Where("item_id = ?", itemId).Order("npc_id ASC, state_id ASC"), page)
	}
}

// getByNpcIdPagedProvider returns a provider that fetches one page of recipe
// entities for the given crafter npcId, scoped to the active tenant, ordered
// by state_id.
func getByNpcIdPagedProvider(npcId uint32, page model.Page) database.EntityProvider[model.Paged[Entity]] {
	return func(db *gorm.DB) model.Provider[model.Paged[Entity]] {
		return database.PagedQuery[Entity](db.Where("npc_id = ?", npcId).Order("state_id ASC"), page)
	}
}

// getAllForTenant returns every recipe row for the active tenant. Used by
// reindex orchestration when the caller wants to compare prior state.
func getAllForTenant(db *gorm.DB) ([]Entity, error) {
	var entities []Entity
	result := db.Find(&entities)
	return entities, result.Error
}
