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
