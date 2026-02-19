package history

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func create(db *gorm.DB) func(tenantId uuid.UUID, accountId uint32, accountName string, ipAddress string, hwid string, success bool, failureReason string) (Model, error) {
	return func(tenantId uuid.UUID, accountId uint32, accountName string, ipAddress string, hwid string, success bool, failureReason string) (Model, error) {
		a := &Entity{
			TenantId:      tenantId,
			AccountId:     accountId,
			AccountName:   accountName,
			IPAddress:     ipAddress,
			HWID:          hwid,
			Success:       success,
			FailureReason: failureReason,
		}

		err := db.Create(a).Error
		if err != nil {
			return Model{}, err
		}

		return Make(*a)
	}
}

func deleteOlderThan(db *gorm.DB) func(cutoff time.Time) error {
	return func(cutoff time.Time) error {
		return db.Where("created_at < ?", cutoff).Delete(&Entity{}).Error
	}
}

func Make(e Entity) (Model, error) {
	return NewBuilder(e.TenantId, e.AccountId, e.AccountName).
		SetId(e.ID).
		SetIPAddress(e.IPAddress).
		SetHWID(e.HWID).
		SetSuccess(e.Success).
		SetFailureReason(e.FailureReason).
		SetCreatedAt(e.CreatedAt).
		Build()
}
