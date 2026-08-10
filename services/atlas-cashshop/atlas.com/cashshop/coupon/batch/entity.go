package batch

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}

// Entity is the coupon_batches table — the grouping created by one bulk
// generation. GeneratedCount always equals RequestedCount on success: the
// generator RETRIES a code collision rather than skipping it (design §8), so a
// short batch is a bug, not an expected outcome.
type Entity struct {
	Id             uuid.UUID `gorm:"primaryKey;type:uuid"`
	TenantId       uuid.UUID `gorm:"not null;index"`
	Description    string    `gorm:"type:text"`
	RequestedCount uint32    `gorm:"not null"`
	GeneratedCount uint32    `gorm:"not null"`
	CreatedAt      time.Time
}

func (e Entity) TableName() string {
	return "coupon_batches"
}

func (e *Entity) BeforeCreate(_ *gorm.DB) (err error) {
	if e.Id == uuid.Nil {
		e.Id = uuid.New()
	}
	return
}

func Make(e Entity) (Model, error) {
	return Model{
		id:             e.Id,
		description:    e.Description,
		requestedCount: e.RequestedCount,
		generatedCount: e.GeneratedCount,
		createdAt:      e.CreatedAt,
	}, nil
}
