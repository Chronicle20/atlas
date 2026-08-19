package parcel

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Create persists a new parcel and returns the stored Model. TenantId is
// injected by the atlas-database tenant:create callback from the context on
// db when the caller has not set it — this administrator must NOT set
// tenant_id itself.
func Create(db *gorm.DB) func(m Model) (Model, error) {
	return func(m Model) (Model, error) {
		e := entityFromModel(m)
		if e.Id == uuid.Nil {
			e.Id = uuid.New()
		}
		err := db.Create(&e).Error
		if err != nil {
			return Model{}, err
		}
		return Make(e)
	}
}

// UpdateStatus transitions a parcel to status and stamps resolvedAt (the
// receive/discard/expire timestamp).
func UpdateStatus(db *gorm.DB) func(id uuid.UUID, status string, resolvedAt time.Time) error {
	return func(id uuid.UUID, status string, resolvedAt time.Time) error {
		return db.Model(&Entity{}).
			Where("id = ?", id).
			Updates(map[string]interface{}{
				"status":      status,
				"resolved_at": resolvedAt,
			}).Error
	}
}

// StampNotified records LastNotified for a batch of parcels — used by the
// mailbox notification sweep (task-24).
func StampNotified(db *gorm.DB) func(ids []uuid.UUID, at time.Time) error {
	return func(ids []uuid.UUID, at time.Time) error {
		if len(ids) == 0 {
			return nil
		}
		return db.Model(&Entity{}).
			Where("id IN ?", ids).
			Update("last_notified", at).Error
	}
}
