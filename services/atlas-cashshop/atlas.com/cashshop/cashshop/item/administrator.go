package item

import (
	"atlas-cashshop/database"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"math/rand"
	"time"
)

func generateUniqueCashId(tenantId uuid.UUID, db *gorm.DB) (int64, error) {
	for {
		cashId := rand.Int63()
		entities, err := byCashIdEntityProvider(tenantId, cashId)(db)()
		if err != nil {
			return 0, err
		}
		if len(entities) == 0 {
			return cashId, nil
		}
	}
}

func create(tenantId uuid.UUID, templateId uint32, commodityId uint32, quantity uint32, purchasedBy uint32, period uint32, hourlyConfig map[uint32]uint32) database.EntityProvider[Entity] {
	return func(db *gorm.DB) model.Provider[Entity] {
		cashId, err := generateUniqueCashId(tenantId, db)
		if err != nil {
			return model.ErrorProvider[Entity](err)
		}

		expiration := CalculateExpiration(period, templateId, hourlyConfig)

		now := time.Now()
		entity := Entity{
			TenantId:    tenantId,
			CashId:      cashId,
			TemplateId:  templateId,
			CommodityId: commodityId,
			Quantity:    quantity,
			Flag:        0, // Default flag value
			PurchasedBy: purchasedBy,
			Expiration:  expiration,
			CreatedAt:   now,
		}

		err = db.Create(&entity).Error
		if err != nil {
			return model.ErrorProvider[Entity](err)
		}

		return model.FixedProvider[Entity](entity)
	}
}

func deleteById(db *gorm.DB, tenantId uuid.UUID, id uint32) error {
	return db.Where("tenant_id = ? AND id = ?", tenantId, id).Delete(&Entity{}).Error
}

// findOrCreateByCashId finds an existing item by cashId, or creates a new one if not found
// This is used for preserving cashId during transfers between inventory and cash shop
func findOrCreateByCashId(tenantId uuid.UUID, cashId int64, templateId uint32, commodityId uint32, quantity uint32, purchasedBy uint32, period uint32, hourlyConfig map[uint32]uint32) database.EntityProvider[Entity] {
	return func(db *gorm.DB) model.Provider[Entity] {
		// First try to find an existing item with this cashId
		entities, err := byCashIdEntityProvider(tenantId, cashId)(db)()
		if err != nil {
			return model.ErrorProvider[Entity](err)
		}

		// If we found an existing item, return it
		if len(entities) > 0 {
			return model.FixedProvider[Entity](entities[0])
		}

		// No existing item found, create a new one
		now := time.Now()
		expiration := CalculateExpiration(period, templateId, hourlyConfig)

		entity := Entity{
			TenantId:    tenantId,
			CashId:      cashId,
			TemplateId:  templateId,
			CommodityId: commodityId,
			Quantity:    quantity,
			Flag:        0, // Default flag value
			PurchasedBy: purchasedBy,
			Expiration:  expiration,
			CreatedAt:   now,
		}

		err = db.Create(&entity).Error
		if err != nil {
			return model.ErrorProvider[Entity](err)
		}

		return model.FixedProvider[Entity](entity)
	}
}
