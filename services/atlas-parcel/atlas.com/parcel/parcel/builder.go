package parcel

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// Builder constructs a Model. Zero value is usable via NewBuilder; Build
// validates before returning.
type Builder struct {
	id                 uuid.UUID
	tenantId           uuid.UUID
	worldId            world.Id
	senderId           uint32
	senderAccountId    uint32
	senderName         string
	recipientId        uint32
	recipientAccountId uint32
	recipientName      string
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

func NewBuilder() *Builder {
	return &Builder{}
}

func (b *Builder) SetId(id uuid.UUID) *Builder {
	b.id = id
	return b
}

// SetTenantId sets the row's tenant — read-path only (Make populates it from
// a persisted Entity). Create's entityFromModel deliberately ignores it; see
// model.go's TenantId() doc comment.
func (b *Builder) SetTenantId(tenantId uuid.UUID) *Builder {
	b.tenantId = tenantId
	return b
}

func (b *Builder) SetWorldId(worldId world.Id) *Builder {
	b.worldId = worldId
	return b
}

func (b *Builder) SetSenderId(senderId uint32) *Builder {
	b.senderId = senderId
	return b
}

func (b *Builder) SetSenderAccountId(senderAccountId uint32) *Builder {
	b.senderAccountId = senderAccountId
	return b
}

func (b *Builder) SetSenderName(senderName string) *Builder {
	b.senderName = senderName
	return b
}

func (b *Builder) SetRecipientId(recipientId uint32) *Builder {
	b.recipientId = recipientId
	return b
}

func (b *Builder) SetRecipientAccountId(recipientAccountId uint32) *Builder {
	b.recipientAccountId = recipientAccountId
	return b
}

func (b *Builder) SetRecipientName(recipientName string) *Builder {
	b.recipientName = recipientName
	return b
}

func (b *Builder) SetMessage(message string) *Builder {
	b.message = message
	return b
}

func (b *Builder) SetMesoAmount(mesoAmount uint32) *Builder {
	b.mesoAmount = mesoAmount
	return b
}

func (b *Builder) SetFeePaid(feePaid uint32) *Builder {
	b.feePaid = feePaid
	return b
}

func (b *Builder) SetItemId(itemId *uint32) *Builder {
	b.itemId = itemId
	return b
}

func (b *Builder) SetItemType(itemType byte) *Builder {
	b.itemType = itemType
	return b
}

func (b *Builder) SetQuantity(quantity uint16) *Builder {
	b.quantity = quantity
	return b
}

func (b *Builder) SetItemSnapshot(itemSnapshot AssetData) *Builder {
	b.itemSnapshot = itemSnapshot
	return b
}

func (b *Builder) SetStatus(status string) *Builder {
	b.status = status
	return b
}

func (b *Builder) SetQuick(quick bool) *Builder {
	b.quick = quick
	return b
}

func (b *Builder) SetReturned(returned bool) *Builder {
	b.returned = returned
	return b
}

func (b *Builder) SetCreatedAt(createdAt time.Time) *Builder {
	b.createdAt = createdAt
	return b
}

func (b *Builder) SetReceivableAt(receivableAt time.Time) *Builder {
	b.receivableAt = receivableAt
	return b
}

func (b *Builder) SetExpiresAt(expiresAt time.Time) *Builder {
	b.expiresAt = expiresAt
	return b
}

func (b *Builder) SetResolvedAt(resolvedAt *time.Time) *Builder {
	b.resolvedAt = resolvedAt
	return b
}

func (b *Builder) SetLastNotified(lastNotified *time.Time) *Builder {
	b.lastNotified = lastNotified
	return b
}

func (b *Builder) validate() error {
	if b.id == uuid.Nil {
		return errors.New("id is required")
	}
	if b.senderId == 0 {
		return errors.New("senderId is required")
	}
	if b.recipientId == 0 {
		return errors.New("recipientId is required")
	}
	if b.status == "" {
		return errors.New("status is required")
	}
	return nil
}

func (b *Builder) Build() (Model, error) {
	if err := b.validate(); err != nil {
		return Model{}, err
	}
	return Model{
		id:                 b.id,
		tenantId:           b.tenantId,
		worldId:            b.worldId,
		senderId:           b.senderId,
		senderAccountId:    b.senderAccountId,
		senderName:         b.senderName,
		recipientId:        b.recipientId,
		recipientAccountId: b.recipientAccountId,
		recipientName:      b.recipientName,
		message:            b.message,
		mesoAmount:         b.mesoAmount,
		feePaid:            b.feePaid,
		itemId:             b.itemId,
		itemType:           b.itemType,
		quantity:           b.quantity,
		itemSnapshot:       b.itemSnapshot,
		status:             b.status,
		quick:              b.quick,
		returned:           b.returned,
		createdAt:          b.createdAt,
		receivableAt:       b.receivableAt,
		expiresAt:          b.expiresAt,
		resolvedAt:         b.resolvedAt,
		lastNotified:       b.lastNotified,
	}, nil
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

// entityFromModel maps a Model back to its persisted Entity shape, for
// Create. TenantId is deliberately left zero — atlas-database's
// tenant:create callback injects it from context when zero (see
// libs/atlas-database/tenant_scope.go), matching every other tenant-scoped
// entity in this repo (frederick, storage).
func entityFromModel(m Model) Entity {
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
