package parcel

import (
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	packetparcel "github.com/Chronicle20/atlas/libs/atlas-packet/parcel"
)

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
	p := packetparcel.NewParcel(WireId(m.Id()), m.SenderName(), m.MesoAmount(), m.CreatedAt(), m.Message())

	if m.ItemId() == nil {
		return p
	}

	item := packetmodel.NewAsset(false, 0, *m.ItemId(), time.Time{})
	if inventory.Type(m.ItemType()) != inventory.TypeValueEquip {
		item = item.SetStackableInfo(uint32(m.Quantity()), 0, 0)
	}
	return p.SetItem(item)
}
