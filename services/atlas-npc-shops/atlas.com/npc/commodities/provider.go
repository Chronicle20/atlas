package commodities

import (
	"atlas-npc/database"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func getByNpcId(tenantId uuid.UUID, npcId uint32) database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		var results []Entity
		err := db.Where(&Entity{TenantId: tenantId, NpcId: npcId}).Find(&results).Error
		if err != nil {
			return model.ErrorProvider[[]Entity](err)
		}
		return model.FixedProvider(results)
	}
}

func getAllByTenant(tenantId uuid.UUID) database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		var results []Entity
		err := db.Where(&Entity{TenantId: tenantId}).Find(&results).Error
		if err != nil {
			return model.ErrorProvider[[]Entity](err)
		}
		return model.FixedProvider(results)
	}
}

// getCommodityIdToNpcIdMap returns a provider that gets a map of commodity ID to NPC ID for a tenant
func getCommodityIdToNpcIdMap(tenantId uuid.UUID) database.EntityProvider[map[uuid.UUID]uint32] {
	return func(db *gorm.DB) model.Provider[map[uuid.UUID]uint32] {
		var results []struct {
			Id    uuid.UUID
			NpcId uint32
		}
		err := db.Table("commodities").
			Select("id, npc_id").
			Where("tenant_id = ?", tenantId).
			Where("deleted_at IS NULL").
			Find(&results).Error
		if err != nil {
			return model.ErrorProvider[map[uuid.UUID]uint32](err)
		}

		// Create a map of commodity ID to NPC ID
		commodityIdToNpcId := make(map[uuid.UUID]uint32)
		for _, result := range results {
			commodityIdToNpcId[result.Id] = result.NpcId
		}

		return model.FixedProvider(commodityIdToNpcId)
	}
}

// existsByNpcId returns a provider that checks if any commodities exist for a given NPC ID
func existsByNpcId(tenantId uuid.UUID, npcId uint32) database.EntityProvider[bool] {
	return func(db *gorm.DB) model.Provider[bool] {
		var count int64
		err := db.Model(&Entity{}).
			Where(&Entity{TenantId: tenantId, NpcId: npcId}).
			Count(&count).Error
		if err != nil {
			return model.ErrorProvider[bool](err)
		}
		return model.FixedProvider(count > 0)
	}
}

// getDistinctNpcIds returns a provider that gets a distinct list of NPC IDs for a tenant
func getDistinctNpcIds(tenantId uuid.UUID) database.EntityProvider[[]uint32] {
	return func(db *gorm.DB) model.Provider[[]uint32] {
		var results []uint32
		err := db.Table("commodities").
			Select("DISTINCT npc_id").
			Where("tenant_id = ?", tenantId).
			Where("deleted_at IS NULL").
			Order("npc_id").
			Pluck("npc_id", &results).Error
		if err != nil {
			return model.ErrorProvider[[]uint32](err)
		}
		return model.FixedProvider(results)
	}
}
