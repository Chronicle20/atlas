package bid

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Migration creates the bids table. It is a brand-new table (no legacy
// primary-key rewrite), so AutoMigrate alone produces the correct surrogate-key
// shape and the composite index declared on the entity tags.
func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&entity{})
}

// entity is the GORM row for an auction bid.
//
// The primary key is a surrogate UUID (Id); business identity is never the key,
// and a (tenant_id, id) unique index keeps the row tenant-scoped — never a
// unique index on tenant_id alone, which would cap a tenant (and an auction) at
// one bid.
//
// One composite index backs the design's hot query:
//   - (tenant_id, listing_id, state) — the bids on an auction by escrow state
type entity struct {
	Id          uuid.UUID `gorm:"column:id;type:uuid;primaryKey;uniqueIndex:idx_bids_tenant_id,priority:2"`
	TenantId    uuid.UUID `gorm:"column:tenant_id;type:uuid;not null;uniqueIndex:idx_bids_tenant_id,priority:1;index:idx_bids_listing_state,priority:1"`
	ListingId   uuid.UUID `gorm:"column:listing_id;type:uuid;not null;index:idx_bids_listing_state,priority:2"`
	BidderId    uint32    `gorm:"column:bidder_id;not null"`
	Amount      uint32    `gorm:"column:amount;not null"`
	EscrowTxnId uuid.UUID `gorm:"column:escrow_txn_id;type:uuid;not null"`
	State       string    `gorm:"column:state;not null;index:idx_bids_listing_state,priority:3"`

	CreatedAt time.Time `gorm:"column:created_at"`
}

func (e entity) TableName() string {
	return "bids"
}
