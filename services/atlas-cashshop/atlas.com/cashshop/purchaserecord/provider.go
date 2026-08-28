package purchaserecord

import (
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

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
