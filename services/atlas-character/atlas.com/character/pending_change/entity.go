package pending_change

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// Request types.
const (
	TypeNameChange    = "NAME_CHANGE"
	TypeWorldTransfer = "WORLD_TRANSFER"
)

// Lifecycle statuses. Transitions out of StatusPending are one-way; a terminal
// record is never reopened (FR-2.2).
const (
	StatusPending   = "PENDING"
	StatusApplied   = "APPLIED"
	StatusCancelled = "CANCELLED"
	StatusRejected  = "REJECTED"
	StatusExpired   = "EXPIRED"
)

// Migration runs AutoMigrate, then creates the two partial unique indexes by
// raw DDL. GORM tags cannot express a WHERE clause, and these indexes are the
// mechanism behind FR-2.3 (one pending request per type) and FR-3.3 (the name
// soft reservation) — not a redundant guard on application code. Idempotent.
func Migration(db *gorm.DB) error {
	if err := db.AutoMigrate(&entity{}); err != nil {
		return err
	}
	stmts := []string{
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_pc_one_pending_per_type
		   ON character_pending_changes (tenant_id, character_id, type)
		   WHERE status = 'PENDING'`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_pc_name_reservation
		   ON character_pending_changes (tenant_id, requested_name_lower)
		   WHERE status = 'PENDING' AND type = 'NAME_CHANGE'`,
	}
	for _, s := range stmts {
		if err := db.Exec(s).Error; err != nil {
			return err
		}
	}
	return nil
}

type entity struct {
	Id          uuid.UUID `gorm:"type:uuid;primaryKey;not null"`
	TenantId    uuid.UUID `gorm:"not null;index:idx_pc_tenant_char,priority:1"`
	CharacterId uint32    `gorm:"not null;index:idx_pc_tenant_char,priority:2"`
	Type        string    `gorm:"not null"`
	Status      string    `gorm:"not null;index:idx_pc_status"`

	// RequestedNameLower is a stored column rather than a functional index on
	// lower(requested_name) so the reservation lookup and idx_pc_name_reservation
	// agree exactly, and the "reserved" check in CheckNameValidity stays a plain
	// equality predicate.
	RequestedName      *string
	RequestedNameLower *string

	DestinationWorldId *world.Id
	SourceWorldId      world.Id `gorm:"not null"`

	// AssetId is null on the cash-shop purchase path, which carries an
	// entitlement reference correlated by TransactionId instead of an asset.
	AssetId *uint32

	Reason        string    `gorm:"not null;default:''"`
	TransactionId uuid.UUID `gorm:"not null"`

	CreatedAt  time.Time `gorm:"not null"`
	ExpiresAt  time.Time `gorm:"not null"`
	ResolvedAt *time.Time
	NotifiedAt *time.Time
}

func (e entity) TableName() string {
	return "character_pending_changes"
}
