package shops

import (
	"atlas-npc/database"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// createShop returns a provider that creates a shop entity
func createShop(tenantId uuid.UUID, npcId uint32, recharger bool) database.EntityProvider[Entity] {
	return func(db *gorm.DB) model.Provider[Entity] {
		entity := Entity{
			Id:        uuid.New(),
			TenantId:  tenantId,
			NpcId:     npcId,
			Recharger: recharger,
		}
		err := db.Create(&entity).Error
		if err != nil {
			return model.ErrorProvider[Entity](err)
		}
		return model.FixedProvider(entity)
	}
}

// updateShop returns a provider that updates a shop entity
func updateShop(tenantId uuid.UUID, npcId uint32, recharger bool) database.EntityProvider[Entity] {
	return func(db *gorm.DB) model.Provider[Entity] {
		var entity Entity
		err := db.Where(&Entity{TenantId: tenantId, NpcId: npcId}).First(&entity).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return createShop(tenantId, npcId, recharger)(db)
			}
			return model.ErrorProvider[Entity](err)
		}

		entity.Recharger = recharger
		err = db.Save(&entity).Error
		if err != nil {
			return model.ErrorProvider[Entity](err)
		}
		return model.FixedProvider(entity)
	}
}

// deleteAllShops returns a provider that deletes all shop entities for a tenant
func deleteAllShops(tenantId uuid.UUID) database.EntityProvider[bool] {
	return func(db *gorm.DB) model.Provider[bool] {
		err := db.Where(&Entity{TenantId: tenantId}).Delete(&Entity{}).Error
		if err != nil {
			return model.ErrorProvider[bool](err)
		}
		return model.FixedProvider(true)
	}
}
