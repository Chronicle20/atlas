package clientbound

import (
	"bytes"
	"testing"
	"time"

	testlog "github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-packet/parcel"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// wantEquipItemBytesV87 hand-builds the wire encoding of a bare equipment
// asset (model.NewAsset(true, 0, 1302000, time.Time{})) under GMS v87, per
// model.Asset.encodeEquipableInfo (libs/atlas-packet/model/asset.go). v87 is
// on the far side of the MajorAtLeast(83) short-slot gate (asset.go:507),
// the MajorAtLeast(79) hammersApplied gate (asset.go:266), and the
// MajorAtLeast(84) nDurability gate (asset.go:260-262) -- same as v84.
// Confirmed directly on this IDB (session c0829805):
// GW_ItemSlotEquip::RawDecode @0x502eac issues three consecutive Decode4
// calls at 0x5030a3/0x5030bd/0x5030d7, storing to this+192/this+204/this+216
// (experience/nDurability/hammersApplied) -- the same three-Decode4 shape as
// v84's GW_ItemSlotEquip::RawDecode (one more Decode4 than v83's two-field
// shape), and asset.go has no MajorAtLeast gate above 84 (`grep -n
// MajorAtLeast libs/atlas-packet/model/asset.go` tops out at 84), so v87's
// asset-encoding path is byte-identical to v84's -- confirmed on this
// version's own IDB rather than assumed from the neighbouring v84 batch.
// Every numeric field on this asset is its zero value, so each gated region
// below is either absent or all-zero-bytes of its fixed width; only the
// fixed byte offsets differ from wantEquipItemBytesV83
// (task-241 Task 28, RULING 22).
func wantEquipItemBytesV87() []byte {
	var b []byte
	b = append(b, 0x01)                                           // cash-type-byte marker (MajorVersion>12)
	b = append(b, 0xf0, 0xdd, 0x13, 0x00)                         // templateId 1302000 LE
	b = append(b, 0x00)                                           // WriteBool(false) -- not cash
	b = append(b, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff) // MsTime(zero) = -1
	b = append(b, 0x00)                                           // slots = 0
	b = append(b, 0x00)                                           // level = 0
	b = append(b, make([]byte, 30)...)                            // 15 zero equipment stat shorts
	b = append(b, 0x00, 0x00)                                     // WriteAsciiString("") -> short length 0
	b = append(b, 0x00, 0x00)                                     // flag short = 0
	b = append(b, 0x00)                                           // levelType = 0
	b = append(b, 0x00)                                           // level = 0
	b = append(b, 0x00, 0x00, 0x00, 0x00)                         // experience = 0
	b = append(b, 0xff, 0xff, 0xff, 0xff)                         // nDurability = -1 (asset.go:260-262, MajorAtLeast(84), ABSENT on v83)
	b = append(b, 0x00, 0x00, 0x00, 0x00)                         // hammersApplied = 0 (asset.go:266, MajorAtLeast(79))
	b = append(b, make([]byte, 8)...)                             // WriteLong(0) trailing buffer
	b = append(b, 0x00, 0x40, 0xe0, 0xfd, 0x3b, 0x37, 0x4f, 0x01) // 94354848000000000
	b = append(b, 0xff, 0xff, 0xff, 0xff)                         // -1
	return b
}

// v87 PARCEL (op 0x153) family verification (task-241 Task 28, batch 5/8,
// session c0829805, GMSv87_4GB.exe.i64):
//
//	CParcelDlg::OnPacket @0x7346db: `iPacket = CInPacket::Decode1(iPacket)`
//	  mode read @0x734700, then a switch with explicit cases 8/23/24/25/26/27
//	  and a default arm that calls the NoticeResult equivalent (sub_734BF3
//	  @0x734bf3) for everything else (9-22, 28) and additionally triggers
//	  CloseParcelDlg (@0x734682) when a1==18 (@0x734796) -- byte-identical
//	  switch shape to the gms_v83/v84 anchors (RULING D), independently
//	  confirmed against THIS IDB rather than leaned on from the pre-existing
//	  parcel.yaml provenance comment.
//	sub_734BF3 (NoticeResult equivalent) @0x734bf3: two nested switches
//	  (a1<=16 @0x734c02, a1>16 @0x734cbe) mapping mode values to StringPool
//	  ids; explicit cases 10/11/12/13/14/16 and default-a1==15 in the first
//	  switch, explicit cases 17/18/21/22 and 19/28-shared (LABEL_25
//	  @0x734d06) in the second. Modes 9 and 20 hit no case in either switch
//	  and fall through with no notice call (bodyless, silent) -- matches the
//	  v83/v84-documented pattern. None of these 9-22,28 arms consume any
//	  extra CInPacket bytes beyond the mode byte in OnPacket's default arm,
//	  so every notice-only arm's wire shape is just the single mode byte.
//	PARCEL::Decode @0x5035ce: CInPacket::DecodeBuffer(0xEA=234) fixed block,
//	  then Decode1(hasItem) and an optional GW_ItemSlotBase::Decode item --
//	  byte-identical shape to v83/v84's PARCEL::Decode.
//	CTabReceive::SetParcel @0x72e688 (called from case 8 via
//	  CParcelDlg::SetParcelDlg @0x734218): Decode1(mailbox count) +
//	  count * PARCEL::Decode, then Decode1(arrived count) +
//	  count * PARCEL::Decode.
//
// Full per-arm addresses and read orders are recorded in the spliced export
// entries (docs/packets/ida-exports/gms_v87.json, keys
// "CParcelDlg::OnPacket#<Arm>") and in the generated audit reports
// (docs/packets/audits/gms_v87/Parcel*.json).

// packet-audit:verify packet=parcel/clientbound/ParcelSendEnableActions version=gms_v87 ida=0x734c02
// packet-audit:verify packet=parcel/clientbound/ParcelNotEnoughMesos version=gms_v87 ida=0x734c9e
// packet-audit:verify packet=parcel/clientbound/ParcelIncorrectRequest version=gms_v87 ida=0x734c88
// packet-audit:verify packet=parcel/clientbound/ParcelNameDoesNotExist version=gms_v87 ida=0x734c72
// packet-audit:verify packet=parcel/clientbound/ParcelSameAccount version=gms_v87 ida=0x734c5c
// packet-audit:verify packet=parcel/clientbound/ParcelReceiverStorageFull version=gms_v87 ida=0x734c46
// packet-audit:verify packet=parcel/clientbound/ParcelReceiverUnableToReceive version=gms_v87 ida=0x734c30
// packet-audit:verify packet=parcel/clientbound/ParcelSenderUniqueConflict version=gms_v87 ida=0x734cb4
// packet-audit:verify packet=parcel/clientbound/ParcelMesoLimit version=gms_v87 ida=0x734d2c
// packet-audit:verify packet=parcel/clientbound/ParcelSuccessfullySent version=gms_v87 ida=0x734796
// packet-audit:verify packet=parcel/clientbound/ParcelUnknownError version=gms_v87 ida=0x734d06
// packet-audit:verify packet=parcel/clientbound/ParcelRecvEnableActions version=gms_v87 ida=0x734cbe
// packet-audit:verify packet=parcel/clientbound/ParcelRecvNoFreeSlots version=gms_v87 ida=0x734cf3
// packet-audit:verify packet=parcel/clientbound/ParcelRecvUniqueConflict version=gms_v87 ida=0x734ce0
// packet-audit:verify packet=parcel/clientbound/ParcelUnknownError2 version=gms_v87 ida=0x734d06
func TestParcelResultArmsV87(t *testing.T) {
	ctx := pt.CreateContext("GMS", 87, 1)

	tests := []struct {
		name string
		mode byte
		got  []byte
	}{
		{"SendEnableActions", 9, NewParcelSendEnableActions(9).Encode(nil, ctx)(nil)},
		{"NotEnoughMesos", 10, NewParcelNotEnoughMesos(10).Encode(nil, ctx)(nil)},
		{"IncorrectRequest", 11, NewParcelIncorrectRequest(11).Encode(nil, ctx)(nil)},
		{"NameDoesNotExist", 12, NewParcelNameDoesNotExist(12).Encode(nil, ctx)(nil)},
		{"SameAccount", 13, NewParcelSameAccount(13).Encode(nil, ctx)(nil)},
		{"ReceiverStorageFull", 14, NewParcelReceiverStorageFull(14).Encode(nil, ctx)(nil)},
		{"ReceiverUnableToReceive", 15, NewParcelReceiverUnableToReceive(15).Encode(nil, ctx)(nil)},
		{"SenderUniqueConflict", 16, NewParcelSenderUniqueConflict(16).Encode(nil, ctx)(nil)},
		{"MesoLimit", 17, NewParcelMesoLimit(17).Encode(nil, ctx)(nil)},
		{"SuccessfullySent", 18, NewParcelSuccessfullySent(18).Encode(nil, ctx)(nil)},
		{"UnknownError", 19, NewParcelUnknownError(19).Encode(nil, ctx)(nil)},
		{"RecvEnableActions", 20, NewParcelRecvEnableActions(20).Encode(nil, ctx)(nil)},
		{"RecvNoFreeSlots", 21, NewParcelRecvNoFreeSlots(21).Encode(nil, ctx)(nil)},
		{"RecvUniqueConflict", 22, NewParcelRecvUniqueConflict(22).Encode(nil, ctx)(nil)},
		{"UnknownError2", 28, NewParcelUnknownError2(28).Encode(nil, ctx)(nil)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			want := []byte{tc.mode}
			if !bytes.Equal(tc.got, want) {
				t.Errorf("%s: got % x want % x", tc.name, tc.got, want)
			}
		})
	}
}

// packet-audit:verify packet=parcel/clientbound/ParcelOpenQuick version=gms_v87 ida=0x7348a4
func TestParcelOpenQuickV87(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 87, 1)
	got := NewParcelOpenQuick(0x1A).Encode(l, ctx)(nil)
	want := []byte{0x1A}
	if !bytes.Equal(got, want) {
		t.Errorf("OpenQuick: got % x want % x", got, want)
	}
}

// packet-audit:verify packet=parcel/clientbound/ParcelParcelRemoved version=gms_v87 ida=0x7349fe
func TestParcelRemovedV87(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 87, 1)
	got := NewParcelRemoved(0x17, 7, ParcelRemovedKindDiscarded).Encode(l, ctx)(nil)
	want := []byte{0x17, 0x07, 0x00, 0x00, 0x00, 0x03}
	if !bytes.Equal(got, want) {
		t.Errorf("ParcelRemoved: got % x want % x", got, want)
	}
}

// packet-audit:verify packet=parcel/clientbound/ParcelParcelArrived version=gms_v87 ida=0x734998
func TestParcelArrivedV87(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 87, 1)
	sentAt := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)
	p := parcel.NewParcel(7, "Alice", 1000, sentAt, "hi")
	pBytes := p.Encode(l, ctx)(nil)

	got := NewParcelArrived(0x18, p).Encode(l, ctx)(nil)
	var want []byte
	want = append(want, 0x18)
	want = append(want, pBytes...)
	if !bytes.Equal(got, want) {
		t.Errorf("ParcelArrived: got % x want % x", got, want)
	}
}

// TestParcelArrivedV87WithItem is a RULING 22 fixture (task-241 Task 28,
// batch 5/8): the embedded Parcel's asset branch (parcel.go:166-170,
// HasItem/WriteBool + conditional item.Encode) is the only version-divergent
// path in the whole PARCEL family (grep -rn 'MajorAtLeast|MajorVersion'
// libs/atlas-packet/parcel/ is empty), and on v87 it is byte-identical to
// v84 (confirmed above via GW_ItemSlotEquip::RawDecode @0x502eac -- same
// three-Decode4 shape, no v87-specific asset gate exists in asset.go).
func TestParcelArrivedV87WithItem(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 87, 1)
	sentAt := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)
	item := model.NewAsset(true, 0, 1302000, time.Time{})
	p := parcel.NewParcel(7, "Alice", 1000, sentAt, "hi").SetItem(item).SetQuick(true)

	name := make([]byte, 13)
	copy(name, "Alice")
	msg := make([]byte, 201)
	copy(msg, "hi")
	filetime := model.MsTimeBytes(sentAt)

	var pBytes []byte
	pBytes = append(pBytes, 0x07, 0x00, 0x00, 0x00)
	pBytes = append(pBytes, name...)
	pBytes = append(pBytes, 0xe8, 0x03, 0x00, 0x00)
	pBytes = append(pBytes, filetime[:]...)
	pBytes = append(pBytes, 0x01, 0x00, 0x00, 0x00) // quick flag LE (quick=true, message non-empty)
	pBytes = append(pBytes, msg...)
	pBytes = append(pBytes, 0x01) // hasItem = true
	pBytes = append(pBytes, wantEquipItemBytesV87()...)

	got := p.Encode(l, ctx)(nil)
	if !bytes.Equal(got, pBytes) {
		t.Fatalf("parcel with item: got % x\nwant % x", got, pBytes)
	}

	gotArrived := NewParcelArrived(0x18, p).Encode(l, ctx)(nil)
	var wantArrived []byte
	wantArrived = append(wantArrived, 0x18)
	wantArrived = append(wantArrived, pBytes...)
	if !bytes.Equal(gotArrived, wantArrived) {
		t.Errorf("ParcelArrived with item: got % x want % x", gotArrived, wantArrived)
	}
}

// packet-audit:verify packet=parcel/clientbound/ParcelAlarmNamed version=gms_v87 ida=0x7348b4
func TestParcelAlarmNamedV87(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 87, 1)
	got := NewParcelAlarmNamed(0x19, "Alice", true).Encode(l, ctx)(nil)
	var want []byte
	want = append(want, 0x19, 0x05, 0x00)
	want = append(want, []byte("Alice")...)
	want = append(want, 0x01)
	if !bytes.Equal(got, want) {
		t.Errorf("AlarmNamed: got % x want % x", got, want)
	}
}

// packet-audit:verify packet=parcel/clientbound/ParcelAlarmGeneric version=gms_v87 ida=0x734808
func TestParcelAlarmGenericV87(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 87, 1)
	got := NewParcelAlarmGeneric(0x1B, true).Encode(l, ctx)(nil)
	want := []byte{0x1B, 0x01}
	if !bytes.Equal(got, want) {
		t.Errorf("AlarmGeneric: got % x want % x", got, want)
	}
}

// packet-audit:verify packet=parcel/clientbound/ParcelOpen version=gms_v87 ida=0x734b43
func TestParcelOpenV87(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 87, 1)
	sentAt := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)
	p := parcel.NewParcel(7, "Alice", 1000, sentAt, "hi")
	pBytes := p.Encode(l, ctx)(nil)

	got := NewParcelOpen(8, true, []parcel.Parcel{p}, []parcel.Parcel{p}).Encode(l, ctx)(nil)
	var want []byte
	want = append(want, 0x08, 0x01, 0x01)
	want = append(want, pBytes...)
	want = append(want, 0x01)
	want = append(want, pBytes...)
	if !bytes.Equal(got, want) {
		t.Errorf("Open: got % x want % x", got, want)
	}
}

// TestParcelOpenV87WithItem is a RULING 22 retro-fit companion to
// TestParcelArrivedV87WithItem: it exercises the same asset-bearing Parcel
// through the OPEN arm's mailbox/arrived slices (CTabReceive::SetParcel
// @0x72e688).
func TestParcelOpenV87WithItem(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 87, 1)
	sentAt := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)
	item := model.NewAsset(true, 0, 1302000, time.Time{})
	p := parcel.NewParcel(7, "Alice", 1000, sentAt, "hi").SetItem(item)

	name := make([]byte, 13)
	copy(name, "Alice")
	msg := make([]byte, 201)
	copy(msg, "hi")
	filetime := model.MsTimeBytes(sentAt)

	var pBytes []byte
	pBytes = append(pBytes, 0x07, 0x00, 0x00, 0x00)
	pBytes = append(pBytes, name...)
	pBytes = append(pBytes, 0xe8, 0x03, 0x00, 0x00)
	pBytes = append(pBytes, filetime[:]...)
	pBytes = append(pBytes, 0x00, 0x00, 0x00, 0x00) // quick flag LE (quick=false, message non-empty)
	pBytes = append(pBytes, msg...)
	pBytes = append(pBytes, 0x01) // hasItem = true
	pBytes = append(pBytes, wantEquipItemBytesV87()...)

	got := NewParcelOpen(8, true, []parcel.Parcel{p}, []parcel.Parcel{p}).Encode(l, ctx)(nil)
	var want []byte
	want = append(want, 0x08, 0x01, 0x01)
	want = append(want, pBytes...)
	want = append(want, 0x01)
	want = append(want, pBytes...)
	if !bytes.Equal(got, want) {
		t.Errorf("Open with item: got % x want % x", got, want)
	}
}
