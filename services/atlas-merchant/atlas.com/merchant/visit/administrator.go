package visit

import (
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// upsertVisit atomically increments a visitor's tally, inserting the row on
// first visit. The unique (tenant, shop, name) index drives the conflict.
func upsertVisit(tenantId, shopId uuid.UUID, name string) database.EntityProvider[bool] {
	return func(db *gorm.DB) model.Provider[bool] {
		e := &Entity{Id: uuid.New(), TenantId: tenantId, ShopId: shopId, Name: name, Count: 1}
		err := db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "tenant_id"}, {Name: "shop_id"}, {Name: "name"}},
			DoUpdates: clause.Assignments(map[string]interface{}{"count": gorm.Expr("merchant_visits.count + 1")}),
		}).Create(e).Error
		if err != nil {
			return model.ErrorProvider[bool](err)
		}
		return model.FixedProvider(true)
	}
}
