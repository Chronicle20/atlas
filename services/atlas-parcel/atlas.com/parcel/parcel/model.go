package parcel

import (
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// Model is the immutable domain representation of a parcel in Duey's
// custody. Constructed only via Builder.Build or Make(Entity) — there is no
// exported constructor that skips validation.
type Model struct {
	id                 uuid.UUID
	worldId            world.Id
	senderId           uint32
	senderAccountId    uint32
	senderName         string
	recipientId        uint32
	recipientAccountId uint32
	message            string
	mesoAmount         uint32
	feePaid            uint32
	itemId             *uint32
	itemType           byte
	quantity           uint16
	itemSnapshot       AssetData
	status             string
	quick              bool
	returned           bool
	createdAt          time.Time
	receivableAt       time.Time
	expiresAt          time.Time
	resolvedAt         *time.Time
	lastNotified       *time.Time
}

func (m Model) Id() uuid.UUID { return m.id }

func (m Model) WorldId() world.Id { return m.worldId }

func (m Model) SenderId() uint32 { return m.senderId }

func (m Model) SenderAccountId() uint32 { return m.senderAccountId }

func (m Model) SenderName() string { return m.senderName }

func (m Model) RecipientId() uint32 { return m.recipientId }

func (m Model) RecipientAccountId() uint32 { return m.recipientAccountId }

func (m Model) Message() string { return m.message }

func (m Model) MesoAmount() uint32 { return m.mesoAmount }

func (m Model) FeePaid() uint32 { return m.feePaid }

func (m Model) ItemId() *uint32 { return m.itemId }

func (m Model) ItemType() byte { return m.itemType }

func (m Model) Quantity() uint16 { return m.quantity }

func (m Model) ItemSnapshot() AssetData { return m.itemSnapshot }

func (m Model) Status() string { return m.status }

func (m Model) Quick() bool { return m.quick }

func (m Model) Returned() bool { return m.returned }

func (m Model) CreatedAt() time.Time { return m.createdAt }

func (m Model) ReceivableAt() time.Time { return m.receivableAt }

func (m Model) ExpiresAt() time.Time { return m.expiresAt }

func (m Model) ResolvedAt() *time.Time { return m.resolvedAt }

func (m Model) LastNotified() *time.Time { return m.lastNotified }
