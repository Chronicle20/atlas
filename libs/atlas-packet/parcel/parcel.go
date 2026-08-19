// Package parcel holds the wire struct shared by the Parcel clientbound
// dispatcher's OPEN/OPEN_QUICK/PARCEL_ARRIVED arms (task-241 design.md §5.3).
package parcel

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

// Parcel models the client's PARCEL struct (PARCEL::Decode @0x4E4345, GMS
// v83). PARCEL::Decode reads a FIXED 234-byte block via a single
// CInPacket::DecodeBuffer(this, 234) call — a raw memcpy, not a
// field-by-field parse — then Decode1 (hasItem bool) and, if set, one
// GW_ItemSlotBase.
//
// Field offsets inside the 234-byte block are established from the three
// downstream consumers of the decoded object (the only xrefs to
// PARCEL::Decode in the v83 IDB: CTabReceive::SetParcel @0x6EF69C, called
// twice, and CParcelDlg::OnPacket @0x6F56EA via CTabReceive::SetParcel):
//
//	+0       uint32 parcelId        CTabReceive::ReceiveParcel encodes
//	                                 *(parcel+0) as the outbound RECEIVE
//	                                 request's id (v83 @0x6F0D66,
//	                                 COutPacket::Encode4); the unnamed
//	                                 sub_6F0DC3 does the same for DISCARD
//	                                 (@0x6F0DC3 region).
//	+4       char[13] senderName    SetParcel formats SP_3878 with
//	                                 ZXString<char>::Format(..., parcel+4)
//	                                 (v83 @0x6EF7A1).
//	+17      uint32 mesos           SetParcel formats SP_3879 gated on
//	                                 *(parcel+17)!=0, value read from the
//	                                 same offset (v83 @0x6EF7E2/@0x6EF801).
//	+21      uint64 sentAt          v72 CTabReceive::ReceiveParcel computes
//	          (FILETIME)             its 30-day eligibility window from
//	                                 *(uint64*)(parcel+21) (design.md §5.3,
//	                                 v72 ReceiveParcel @0x65AF41 region).
//	                                 Corroborated directly in v83:
//	                                 CTabReceive::ReceiveParcel @0x6F0D11
//	                                 computes the same <30-day check off
//	                                 *(parcel+21).
//	+29..233 message + padding      No client consumer in the v83 IDB reads
//	          (205 bytes)            this span field-by-field — SetParcel and
//	                                 OnPacket only ever dereference offsets
//	                                 0, 4, 17, 21 (and, on the C++ object,
//	                                 238, unrelated to the wire buffer) of
//	                                 the decoded PARCEL; the 234-byte block
//	                                 itself is copied opaquely by
//	                                 DecodeBuffer. The exact message/padding
//	                                 boundary within this span is therefore
//	                                 NOT independently decompile-confirmed.
//	                                 Modeled per design.md §5.3 ("message
//	                                 lives inside it") as a single
//	                                 zero-padded ASCII buffer, using the same
//	                                 fixed-raw-buffer encoding style as the
//	                                 confirmed senderName[13] neighbor field;
//	                                 this is a design-level inference, not an
//	                                 IDA-cited fact — flagged here rather
//	                                 than asserted as verified. The 205-byte
//	                                 width itself IS derived: DecodeBuffer's
//	                                 total is 234, and the four confirmed
//	                                 leading fields consume exactly
//	                                 4+13+4+8 = 29 bytes, leaving 234-29=205.
//	+234     bool hasItem           CInPacket::Decode1 call immediately
//	                                 after the 234-byte DecodeBuffer, inside
//	                                 PARCEL::Decode itself (v83 @0x4E4365).
//	+235..   GW_ItemSlotBase item    GW_ItemSlotBase::Decode, only when
//	          (optional)             hasItem is set (v83 @0x4E4378);
//	                                 encoded here via the existing
//	                                 version-aware model.Asset codec.
const (
	parcelSenderNameWidth = 13
	parcelMessageWidth    = 205
)

// Parcel is one mailbox/arrival entry.
type Parcel struct {
	id         uint32
	senderName string
	mesos      uint32
	sentAt     time.Time
	message    string
	item       *model.Asset
}

// NewParcel constructs a Parcel with no attached item. Use SetItem to attach
// one.
func NewParcel(id uint32, senderName string, mesos uint32, sentAt time.Time, message string) Parcel {
	return Parcel{id: id, senderName: senderName, mesos: mesos, sentAt: sentAt, message: message}
}

func (p Parcel) Id() uint32         { return p.id }
func (p Parcel) SenderName() string { return p.senderName }
func (p Parcel) Mesos() uint32      { return p.mesos }
func (p Parcel) SentAt() time.Time  { return p.sentAt }
func (p Parcel) Message() string    { return p.message }
func (p Parcel) HasItem() bool      { return p.item != nil }

// Item returns the attached asset and whether one is set.
func (p Parcel) Item() (model.Asset, bool) {
	if p.item == nil {
		return model.Asset{}, false
	}
	return *p.item, true
}

// SetItem returns a copy of p with the given asset attached.
func (p Parcel) SetItem(a model.Asset) Parcel {
	p.item = &a
	return p
}

// writeFixedAscii writes s as raw ASCII bytes into a fixed-width buffer,
// zero-padded (or truncated) to exactly width bytes — the encoding style
// PARCEL::Decode's raw DecodeBuffer copy requires for its fixed block.
func writeFixedAscii(w *response.Writer, s string, width int) {
	b := make([]byte, width)
	copy(b, s)
	w.WriteByteArray(b)
}

// Encode writes the 234-byte fixed block, the hasItem bool, and (if set)
// the GW_ItemSlotBase item.
func (p Parcel) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(p.id)
		writeFixedAscii(w, p.senderName, parcelSenderNameWidth)
		w.WriteInt(p.mesos)
		sentAt := model.MsTimeBytes(p.sentAt)
		w.WriteByteArray(sentAt[:])
		writeFixedAscii(w, p.message, parcelMessageWidth)
		w.WriteBool(p.HasItem())
		if p.item != nil {
			item := *p.item
			w.WriteByteArray(item.Encode(l, ctx)(options))
		}
		return w.Bytes()
	}
}
