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

func createEntityProvider(tenantId uuid.UUID, templateId uint32, quantity uint32, purchasedBy uint32) database.EntityProvider[Entity] {
	return func(db *gorm.DB) model.Provider[Entity] {
		cashId, err := generateUniqueCashId(tenantId, db)
		if err != nil {
			return model.ErrorProvider[Entity](err)
		}

		expiration := time.Now().AddDate(0, 0, 30) // 30 days from now

		entity := Entity{
			TenantId:    tenantId,
			CashId:      cashId,
			TemplateId:  templateId,
			Quantity:    quantity,
			Flag:        0, // Default flag value
			PurchasedBy: purchasedBy,
			Expiration:  expiration,
		}

		err = db.Create(&entity).Error
		if err != nil {
			return model.ErrorProvider[Entity](err)
		}

		return model.FixedProvider[Entity](entity)
	}
}
