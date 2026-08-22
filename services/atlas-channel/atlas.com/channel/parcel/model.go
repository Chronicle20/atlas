package parcel

import (
	"encoding/binary"
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	packetparcel "github.com/Chronicle20/atlas/libs/atlas-packet/parcel"
)

// Model is atlas-channel's read-only view of one parcel in Duey's custody.
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
	status             string
	quick              bool
	returned           bool
	createdAt          time.Time
	receivableAt       time.Time
	expiresAt          time.Time
	lastNotified       *time.Time
}

func (m Model) Id() uuid.UUID              { return m.id }
func (m Model) WorldId() world.Id          { return m.worldId }
func (m Model) SenderId() uint32           { return m.senderId }
func (m Model) SenderAccountId() uint32    { return m.senderAccountId }
func (m Model) SenderName() string         { return m.senderName }
func (m Model) RecipientId() uint32        { return m.recipientId }
func (m Model) RecipientAccountId() uint32 { return m.recipientAccountId }
func (m Model) Message() string            { return m.message }
func (m Model) MesoAmount() uint32         { return m.mesoAmount }
func (m Model) FeePaid() uint32            { return m.feePaid }
func (m Model) ItemId() *uint32            { return m.itemId }
func (m Model) ItemType() byte             { return m.itemType }
func (m Model) Quantity() uint16           { return m.quantity }
func (m Model) Status() string             { return m.status }
func (m Model) Quick() bool                { return m.quick }
func (m Model) Returned() bool             { return m.returned }
func (m Model) CreatedAt() time.Time       { return m.createdAt }
func (m Model) ReceivableAt() time.Time    { return m.receivableAt }
func (m Model) ExpiresAt() time.Time       { return m.expiresAt }
func (m Model) LastNotified() *time.Time   { return m.lastNotified }

// WireId projects a parcel's atlas-parcel uuid.UUID identity onto the
// wire's uint32 parcelId — the client's PARCEL struct's `+0 uint32 parcelId`,
// echoed verbatim by CTabReceive::ReceiveParcel/DiscardParcel (design §5.3).
// It resolves the client's wire-format uint32 parcelId against atlas-parcel's
// uuid.UUID row identity by taking the first 4 bytes of the id, big-endian.
// This is a deliberate, self-contained engineering choice for a wire detail
// the design doc explicitly leaves to implementation ("the exact byte
// layout... is derived during implementation", §5.3). Every caller that
// projects a parcel id onto the wire — DUEY_ACTION RECEIVE/DISCARD's
// resolution and Model.ToPacket()'s emission of the OPEN list body alike —
// MUST use this function rather than re-deriving the projection, or the
// two directions will silently disagree.
func WireId(id uuid.UUID) uint32 {
	return binary.BigEndian.Uint32(id[:4])
}

// ToPacket maps m onto the client's PARCEL wire struct (libs/atlas-packet/
// parcel.Parcel, design §5.3) for the OPEN/OPEN_QUICK mailbox lists.
//
// Id is projected through WireId — the SAME 4-byte big-endian truncation
// DUEY_ACTION RECEIVE/DISCARD resolve back against (see WireId's doc
// comment). Every caller that puts a parcel id on the wire MUST go through
// it, or the two directions silently disagree.
//
// The attached item, when present, carries only id, type and quantity —
// this Model's fields (RestModel's doc comment) do not carry atlas-parcel's
// internal equip-stat item snapshot (AssetData), which is the saga
// expansion's concern, never atlas-channel's read-only view. An equipped
// item therefore shows on the wire with zero-valued equip stats rather than
// invented ones — flagged here rather than silently guessed.
func (m Model) ToPacket() packetparcel.Parcel {
	p := packetparcel.NewParcel(WireId(m.Id()), m.SenderName(), m.MesoAmount(), m.ExpiresAt(), m.Message())

	if m.ItemId() == nil {
		return p
	}

	item := packetmodel.NewAsset(true, 0, *m.ItemId(), time.Time{})
	if inventory.Type(m.ItemType()) != inventory.TypeValueEquip {
		item = item.SetStackableInfo(uint32(m.Quantity()), 0, 0)
	}
	return p.SetItem(item)
}
