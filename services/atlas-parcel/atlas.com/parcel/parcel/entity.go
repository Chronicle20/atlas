package parcel

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Status values for a parcel's custody lifecycle.
const (
	StatusPending   = "pending"
	StatusReceived  = "received"
	StatusDiscarded = "discarded"
	StatusExpired   = "expired"
)

// ReceivableDelay is how long a normal (non-return-leg) parcel sits in
// transit before it becomes receivable at the recipient's post office.
const ReceivableDelay = 24 * time.Hour

// ExpiryWindow is how long an unreceived parcel remains claimable before it
// is swept and returned/discarded.
const ExpiryWindow = 30 * 24 * time.Hour

// Entity is the persisted record of a parcel in Duey's custody.
type Entity struct {
	gorm.Model
	Id       uuid.UUID `gorm:"type:uuid;primaryKey"`
	TenantId uuid.UUID `gorm:"type:uuid;not null;index:idx_parcels_recipient,priority:1;index:idx_parcels_sender,priority:1;index:idx_parcels_sweep,priority:1"`
	WorldId  byte      `gorm:"not null"`

	SenderId        uint32 `gorm:"not null;index:idx_parcels_sender,priority:2"`
	SenderAccountId uint32 `gorm:"not null"`
	SenderName      string `gorm:"not null"`

	RecipientId        uint32 `gorm:"not null;index:idx_parcels_recipient,priority:2"`
	RecipientAccountId uint32 `gorm:"not null"`

	Message    string
	MesoAmount uint32
	FeePaid    uint32

	ItemId       *uint32
	ItemType     byte
	Quantity     uint16
	ItemSnapshot AssetData `gorm:"type:jsonb"`

	Status   string `gorm:"not null;index:idx_parcels_recipient,priority:3;index:idx_parcels_sender,priority:3;index:idx_parcels_sweep,priority:2"`
	Quick    bool   `gorm:"not null"`
	Returned bool   `gorm:"not null"`

	CreatedAt    time.Time `gorm:"not null"`
	ReceivableAt time.Time `gorm:"not null"`
	ExpiresAt    time.Time `gorm:"not null;index:idx_parcels_sweep,priority:3"`
	ResolvedAt   *time.Time
	LastNotified *time.Time
}

func (e *Entity) TableName() string {
	return "parcels"
}

// Migration creates/updates the parcels table.
func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}
