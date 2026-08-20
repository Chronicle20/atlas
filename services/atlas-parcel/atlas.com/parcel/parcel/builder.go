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
// a persisted Entity). Create's Model.ToEntity deliberately ignores it; see
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
