package asset

import (
	"math/rand"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
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

func create(db *gorm.DB, tenantId uuid.UUID, compartmentId uuid.UUID, templateId uint32, commodityId uint32, currency uint32, quantity uint32, petId uint32, purchasedBy uint32, expiration time.Time, giftFrom string, giftMessage string) model.Provider[Entity] {
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
		Currency:      currency,
		Quantity:      quantity,
		Flag:          0,
		PetId:         petId,
		PurchasedBy:   purchasedBy,
		Expiration:    expiration,
		CreatedAt:     time.Now(),
		GiftFrom:      giftFrom,
		GiftMessage:   giftMessage,
	}

	if err := db.Create(&entity).Error; err != nil {
		return model.ErrorProvider[Entity](err)
	}

	return model.FixedProvider(entity)
}

func findOrCreateByCashId(db *gorm.DB, tenantId uuid.UUID, cashId int64, compartmentId uuid.UUID, templateId uint32, commodityId uint32, currency uint32, quantity uint32, petId uint32, purchasedBy uint32, expiration time.Time) model.Provider[Entity] {
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
		Currency:      currency,
		Quantity:      quantity,
		Flag:          0,
		PetId:         petId,
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

// updateGiftAcknowledged marks every asset in compartmentId whose CashId
// appears in cashIds as presented (task-240 Defect H). Scoped by
// compartmentId as well as CashId so a caller cannot accidentally drain an
// asset in a compartment it never resolved from the requesting account. A
// nil/empty cashIds is a no-op, not an unbounded update.
func updateGiftAcknowledged(db *gorm.DB, compartmentId uuid.UUID, cashIds []int64) error {
	if len(cashIds) == 0 {
		return nil
	}
	return db.Model(&Entity{}).
		Where("compartment_id = ? AND cash_id IN ?", compartmentId, cashIds).
		Update("gift_acknowledged", true).Error
}

// updateGiftNoteSent marks the asset in compartmentId whose CashId equals
// cashId as having had its gift-forward note sent (task-240 Defect I).
// Scoped by compartmentId as well as CashId, same rationale as
// updateGiftAcknowledged. A cashId that does not resolve to any row in this
// compartment updates nothing and is not an error.
func updateGiftNoteSent(db *gorm.DB, compartmentId uuid.UUID, cashId int64) error {
	return db.Model(&Entity{}).
		Where("compartment_id = ? AND cash_id = ?", compartmentId, cashId).
		Update("gift_note_sent", true).Error
}
