package recipe

import (
	"gorm.io/gorm"
)

// getByItemIdProvider returns a provider that fetches recipe entities for the
// given output itemId, scoped to the active tenant via GORM tenant callbacks,
// ordered by (npc_id, state_id) for deterministic responses.
func getByItemIdProvider(itemId uint32) func(db *gorm.DB) func() ([]Entity, error) {
	return func(db *gorm.DB) func() ([]Entity, error) {
		return func() ([]Entity, error) {
			var entities []Entity
			result := db.Where("item_id = ?", itemId).Order("npc_id ASC, state_id ASC").Find(&entities)
			return entities, result.Error
		}
	}
}

// getByNpcIdProvider returns a provider that fetches recipe entities for the
// given crafter npcId, scoped to the active tenant, ordered by state_id.
func getByNpcIdProvider(npcId uint32) func(db *gorm.DB) func() ([]Entity, error) {
	return func(db *gorm.DB) func() ([]Entity, error) {
		return func() ([]Entity, error) {
			var entities []Entity
			result := db.Where("npc_id = ?", npcId).Order("state_id ASC").Find(&entities)
			return entities, result.Error
		}
	}
}

// getAllForTenant returns every recipe row for the active tenant. Used by
// reindex orchestration when the caller wants to compare prior state.
func getAllForTenant(db *gorm.DB) ([]Entity, error) {
	var entities []Entity
	result := db.Find(&entities)
	return entities, result.Error
}
