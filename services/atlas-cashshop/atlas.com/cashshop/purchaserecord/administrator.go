package purchaserecord

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Record upserts one purchase. It is called INSIDE the purchase transaction,
// so it takes the tx handle rather than the processor's own db.
//
// The upsert is a single statement on the (tenant_id, account_id,
// serial_number) unique index -- Count = Count + 1, LastAt = now() -- so two
// concurrent inserts for the same purchase cannot both land as separate rows.
func Record(db *gorm.DB, tenantId uuid.UUID, accountId uint32, serialNumber uint32) error {
	now := time.Now()
	e := entity{
		Id:           uuid.New(),
		TenantId:     tenantId,
		AccountId:    accountId,
		SerialNumber: serialNumber,
		Count:        1,
		FirstAt:      now,
		LastAt:       now,
	}
	return db.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "tenant_id"},
			{Name: "account_id"},
			{Name: "serial_number"},
		},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"count":   gorm.Expr("count + 1"),
			"last_at": now,
		}),
	}).Create(&e).Error
}

// Get answers "has this account ever bought this serial number", and how many
// times. A miss is (0, nil) -- not an error.
func Get(db *gorm.DB, tenantId uuid.UUID, accountId uint32, serialNumber uint32) (uint32, error) {
	var e entity
	err := db.Where("tenant_id = ? AND account_id = ? AND serial_number = ?", tenantId, accountId, serialNumber).First(&e).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, err
	}
	return e.Count, nil
}
