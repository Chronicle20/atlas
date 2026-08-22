package parcel

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
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
// is swept and returned/discarded. This is 29 days, not the naively
// "obvious" 30 — the client's own receive guard (CTabReceive::ReceiveParcel,
// v72 @0x65AF41 / v83 @0x6F0D11) computes an UNSIGNED 64-bit quotient of
// (wire +21 deadline - now) / 1 day and refuses the receive unless that
// quotient is strictly < 30. A 30-day window is survivable on the normal
// delivery path (ReceivableAt = CreatedAt + 24h already leaves ~29 days of
// life once a parcel becomes receivable), but NOT on the expiry sweep's
// return leg (task-23), which has ReceivableAt == CreatedAt (no 24h delay):
// exactly 30 days of remaining life at the moment it becomes receivable,
// and floor(30) < 30 is false — the client would refuse it. 29 days keeps
// the quotient at 29 or below for the whole life of every parcel, return
// legs included. See docs/tasks/task-241-duey-parcel-delivery/context.md
// §11 (RISK-4 resolution) for the full derivation.
const ExpiryWindow = 29 * 24 * time.Hour

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
	// RecipientName is populated by the send saga end-to-end (the channel's
	// buildParcelSendSaga sets it from the resolved recipient's Name(); it
	// rides libs/atlas-saga.TransferToParcelPayload/AcceptToParcelPayload,
	// the orchestrator's TransferToParcel expansion, and atlas-parcel's own
	// custody.AcceptToParcelCommandBody unchanged onto this column) — it
	// exists for the expiry sweep's return leg (task-23, design §7.4), which
	// needs the original recipient's display name for the returned parcel's
	// SenderName.
	RecipientName string `gorm:"not null;default:''"`

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

// Make converts a persisted Entity to a Model. Entities read back from the
// database are trusted; a validation failure here indicates a corrupt row
// rather than caller error.
func Make(e Entity) (Model, error) {
	return NewBuilder().
		SetId(e.Id).
		SetTenantId(e.TenantId).
		SetWorldId(world.Id(e.WorldId)).
		SetSenderId(e.SenderId).
		SetSenderAccountId(e.SenderAccountId).
		SetSenderName(e.SenderName).
		SetRecipientId(e.RecipientId).
		SetRecipientAccountId(e.RecipientAccountId).
		SetRecipientName(e.RecipientName).
		SetMessage(e.Message).
		SetMesoAmount(e.MesoAmount).
		SetFeePaid(e.FeePaid).
		SetItemId(e.ItemId).
		SetItemType(e.ItemType).
		SetQuantity(e.Quantity).
		SetItemSnapshot(e.ItemSnapshot).
		SetStatus(e.Status).
		SetQuick(e.Quick).
		SetReturned(e.Returned).
		SetCreatedAt(e.CreatedAt).
		SetReceivableAt(e.ReceivableAt).
		SetExpiresAt(e.ExpiresAt).
		SetResolvedAt(e.ResolvedAt).
		SetLastNotified(e.LastNotified).
		Build()
}

// ToEntity maps a Model back to its persisted Entity shape, for Create.
// TenantId is deliberately left zero — atlas-database's tenant:create
// callback injects it from context when zero (see
// libs/atlas-database/tenant_scope.go), matching every other tenant-scoped
// entity in this repo (frederick, storage). See model.go's TenantId() doc
// comment.
func (m Model) ToEntity() Entity {
	return Entity{
		Id:                 m.id,
		WorldId:            byte(m.worldId),
		SenderId:           m.senderId,
		SenderAccountId:    m.senderAccountId,
		SenderName:         m.senderName,
		RecipientId:        m.recipientId,
		RecipientAccountId: m.recipientAccountId,
		RecipientName:      m.recipientName,
		Message:            m.message,
		MesoAmount:         m.mesoAmount,
		FeePaid:            m.feePaid,
		ItemId:             m.itemId,
		ItemType:           m.itemType,
		Quantity:           m.quantity,
		ItemSnapshot:       m.itemSnapshot,
		Status:             m.status,
		Quick:              m.quick,
		Returned:           m.returned,
		CreatedAt:          m.createdAt,
		ReceivableAt:       m.receivableAt,
		ExpiresAt:          m.expiresAt,
		ResolvedAt:         m.resolvedAt,
		LastNotified:       m.lastNotified,
	}
}
