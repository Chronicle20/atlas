package asset

import (
	"math/rand"
	"time"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func generateUniqueCashId(db *gorm.DB) (int64, error) {
	for {
		cashId := rand.Int63()
		entities, err := byCashIdProvider(cashId)(db)()
		if err != nil {
			return 0, err
		}
		if len(entities) == 0 {
			return cashId, nil
		}
	}
}

func create(db *gorm.DB, tenantId uuid.UUID, compartmentId uuid.UUID, templateId uint32, commodityId uint32, quantity uint32, purchasedBy uint32, expiration time.Time) model.Provider[Entity] {
	cashId, err := generateUniqueCashId(db)
	if err != nil {
		return model.ErrorProvider[Entity](err)
	}

	entity := Entity{
		TenantId:      tenantId,
		CompartmentId: compartmentId,
		CashId:        cashId,
		TemplateId:    templateId,
		CommodityId:   commodityId,
		Quantity:      quantity,
		Flag:          0,
		PurchasedBy:   purchasedBy,
		Expiration:    expiration,
		CreatedAt:     time.Now(),
	}

	if err := db.Create(&entity).Error; err != nil {
		return model.ErrorProvider[Entity](err)
	}

	return model.FixedProvider(entity)
}

func findOrCreateByCashId(db *gorm.DB, tenantId uuid.UUID, cashId int64, compartmentId uuid.UUID, templateId uint32, commodityId uint32, quantity uint32, purchasedBy uint32, expiration time.Time) model.Provider[Entity] {
	entities, err := byCashIdProvider(cashId)(db)()
	if err != nil {
		return model.ErrorProvider[Entity](err)
	}

	if len(entities) > 0 {
		return model.FixedProvider(entities[0])
	}

	entity := Entity{
		TenantId:      tenantId,
		CompartmentId: compartmentId,
		CashId:        cashId,
		TemplateId:    templateId,
		CommodityId:   commodityId,
		Quantity:      quantity,
		Flag:          0,
		PurchasedBy:   purchasedBy,
		Expiration:    expiration,
		CreatedAt:     time.Now(),
	}

	if err := db.Create(&entity).Error; err != nil {
		return model.ErrorProvider[Entity](err)
	}

	return model.FixedProvider(entity)
}

func deleteById(db *gorm.DB, id uint32) error {
	return db.Where("id = ?", id).Delete(&Entity{}).Error
}

func updateQuantity(db *gorm.DB, id uint32, quantity uint32) error {
	return db.Model(&Entity{}).Where("id = ?", id).Update("quantity", quantity).Error
}
