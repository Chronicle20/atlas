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
//	+21      uint64 expiresAt       v72 CTabReceive::ReceiveParcel @0x65AF41,
//	          (FILETIME)             verbatim: `sub ecx,[+21.lo]; sbb
//	                                 eax,[+21.hi]` (parcel[+21] - now, 64-bit),
//	                                 then `call __aulldiv` against
//	                                 0xC92A69C000 (864000000000 = 1 day in
//	                                 100ns ticks), `cmp eax,1Eh` (30),
//	                                 `jl short loc_65AFDF` taken -> builds and
//	                                 sends the RECEIVE request; falling
//	                                 through goes to StringPool id 3864 via
//	                                 CUtilDlg::Notice, a refusal dialog with no
//	                                 packet sent. v83 is byte-identical in
//	                                 shape at @0x6F0D11: `cmp eax,1Eh; jl short
//	                                 loc_6F0D41` -> COutPacket(0x41),
//	                                 Encode1(4), Encode4(*parcelId),
//	                                 SendPacket. v83's divide helper
//	                                 (sub_A62970) is unnamed in that IDB but
//	                                 was confirmed to be __aulldiv by reading
//	                                 it directly: 0xA62984/0xA6298C are both
//	                                 unsigned `div ecx` with no sign-handling
//	                                 prologue.
//	                                 The divide is UNSIGNED, so this is NOT a
//	                                 "how many days old is this parcel"
//	                                 eligibility window — it is a countdown to
//	                                 a future deadline. If *(parcel+21) were in
//	                                 the past (e.g. the parcel's send time),
//	                                 parcel[+21]-now underflows to ~2^64 and
//	                                 the unsigned quotient lands around
//	                                 21,350,398, never < 30: the client
//	                                 refuses every receive. *(parcel+21) must
//	                                 therefore be a timestamp STRICTLY IN THE
//	                                 FUTURE, less than 30 days out — the
//	                                 parcel's expiry deadline, not the time it
//	                                 was sent (task-241 Task 23 / RISK-4
//	                                 resolution, docs/tasks/
//	                                 task-241-duey-parcel-delivery/context.md
//	                                 §11).
//	+29      uint32 hasMessage      CTabReceive::Draw @0x6EFA1F is the ONLY
//	          (flag)                 consumer that reads the tail of the
//	                                 234-byte block. Row rendering
//	                                 (@0x6EFF31/@0x6EFF78) draws a static
//	                                 note-marker string only when
//	                                 `*(parcel + 29)` is non-zero; the detail
//	                                 pane (@0x6F07AB) does a 32-bit compare
//	                                 `cmp [eax+1Dh], edi` (edi == 0) on the
//	                                 same offset to decide whether to clear
//	                                 the note control or read the note text.
//	                                 When the flag is set but the string at
//	                                 +33 is empty, the client substitutes
//	                                 StringPool 3886 (@0x6F07C1) — so this
//	                                 flag must track "message is non-empty",
//	                                 not "a message field exists".
//	+33      char[201] message      NUL-terminated ASCII, read via
//	          (201 bytes)            `ZXString<char>::GetBuffer(..., parcel +
//	                                 33, -1)` @0x6F0801 and passed to
//	                                 sub_6F5D37 (@0x6F080C) to populate the
//	                                 note text control. Width is derived:
//	                                 DecodeBuffer's total is 234, and the
//	                                 leading fields (4+13+4+8 = 29 bytes) plus
//	                                 the +29 flag (4 bytes) consume 33 bytes,
//	                                 leaving 234-33=201.
//	+234     bool hasItem           CInPacket::Decode1 call immediately
//	                                 after the 234-byte DecodeBuffer, inside
//	                                 PARCEL::Decode itself (v83 @0x4E4365).
//	+235..   GW_ItemSlotBase item    GW_ItemSlotBase::Decode, only when
//	          (optional)             hasItem is set (v83 @0x4E4378);
//	                                 encoded here via the existing
//	                                 version-aware model.Asset codec. No slot
//	                                 prefix: GW_ItemSlotBase::Decode
//	                                 @0x4E33F9's first two statements are
//	                                 `v2 = CInPacket::Decode1(a2);
//	                                 GW_ItemSlotBase::CreateItem(&v6, v2)` —
//	                                 the very first byte read is the item
//	                                 TYPE, unlike the inventory/storage call
//	                                 sites where the caller reads a slot
//	                                 byte before invoking Decode. SetItem
//	                                 therefore forces zero-position on every
//	                                 attached asset.
const (
	parcelSenderNameWidth = 13
	parcelMessageWidth    = 201
)

// Parcel is one mailbox/arrival entry.
type Parcel struct {
	id         uint32
	senderName string
	mesos      uint32
	expiresAt  time.Time
	message    string
	item       *model.Asset
}

// NewParcel constructs a Parcel with no attached item. Use SetItem to attach
// one. expiresAt MUST be a future deadline, not the parcel's creation time —
// see the +21 field doc above.
func NewParcel(id uint32, senderName string, mesos uint32, expiresAt time.Time, message string) Parcel {
	return Parcel{id: id, senderName: senderName, mesos: mesos, expiresAt: expiresAt, message: message}
}

func (p Parcel) Id() uint32           { return p.id }
func (p Parcel) SenderName() string   { return p.senderName }
func (p Parcel) Mesos() uint32        { return p.mesos }
func (p Parcel) ExpiresAt() time.Time { return p.expiresAt }
func (p Parcel) Message() string      { return p.message }
func (p Parcel) HasItem() bool        { return p.item != nil }

// Item returns the attached asset and whether one is set.
func (p Parcel) Item() (model.Asset, bool) {
	if p.item == nil {
		return model.Asset{}, false
	}
	return *p.item, true
}

// SetItem returns a copy of p with the given asset attached. The asset is
// always normalized to zero-position before attachment — see the +235..
// field doc above: GW_ItemSlotBase::Decode reads the item TYPE byte first,
// with no leading slot prefix, so the PARCEL wire form can never carry a
// slot, regardless of what the caller passed in.
func (p Parcel) SetItem(a model.Asset) Parcel {
	a = a.SetZeroPosition(true)
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
		expiresAt := model.MsTimeBytes(p.expiresAt)
		w.WriteByteArray(expiresAt[:])
		if p.message != "" {
			w.WriteInt(uint32(1))
		} else {
			w.WriteInt(uint32(0))
		}
		writeFixedAscii(w, p.message, parcelMessageWidth)
		w.WriteBool(p.HasItem())
		if p.item != nil {
			item := *p.item
			w.WriteByteArray(item.Encode(l, ctx)(options))
		}
		return w.Bytes()
	}
}
