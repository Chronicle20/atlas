package opening

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&entity{})
}

// entity is the Cash Shop Surprise open ledger. One row per successfully
// committed open, keyed by the transaction id atlas-channel mints per click.
// Its insert is the FIRST statement in the open transaction, so a Kafka
// redelivery of the same command hits the primary-key violation and the
// whole transaction aborts without granting anything (task-207 FR-4.4).
//
// A real ledger row rather than a compare-and-set on the box quantity: a CAS
// would still consume a SECOND box on redelivery when the player holds a
// stack.
type entity struct {
	TenantId      uuid.UUID `gorm:"primaryKey;not null"`
	TransactionId uuid.UUID `gorm:"primaryKey;not null"`
	AccountId     uint32    `gorm:"not null"`
	AssetId       uint32    `gorm:"not null"`
	CreatedAt     time.Time `gorm:"not null"`
}

func (e entity) TableName() string {
	return "cash_surprise_openings"
}
