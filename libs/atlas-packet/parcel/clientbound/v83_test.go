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

// wantEquipItemBytesV83 hand-builds the wire encoding of a bare equipment
// asset (model.NewAsset(false, 1, 1302000, time.Time{})) under GMS v83, per
// model.Asset.encodeEquipableInfo (libs/atlas-packet/model/asset.go). v83 is
// on the far side of both the MajorAtLeast(83) short-slot gate (asset.go:507)
// and the MajorAtLeast(79) hammersApplied gate (asset.go:266): slot encodes
// as a short (2 bytes) and hammersApplied is written. Every numeric field on
// this asset is its zero value, so each gated region below is either absent
// or all-zero-bytes of its fixed width; only the fixed byte offsets differ
// per version (RULING 22 retro-fit, task-241 Task 28).
func wantEquipItemBytesV83() []byte {
	var b []byte
	b = append(b, 0x01, 0x00)                                     // encodeSlot: short slot (MajorAtLeast(83) true, asset.go:507)
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
	b = append(b, 0x00, 0x00, 0x00, 0x00)                         // hammersApplied = 0 (asset.go:266, MajorAtLeast(79))
	b = append(b, make([]byte, 8)...)                             // WriteLong(0) trailing buffer
	b = append(b, 0x00, 0x40, 0xe0, 0xfd, 0x3b, 0x37, 0x4f, 0x01) // 94354848000000000
	b = append(b, 0xff, 0xff, 0xff, 0xff)                         // -1
	return b
}

// v83 PARCEL (op 0x142) family verification — gms_v83 IS THE ANCHOR (task-241
// Task 28, session 41f09cce, MapleStory_dump.exe.i64). Every byte below is
// derived directly from THIS IDB's own decompile, not cross-version equality
// against another anchor:
//
//	CParcelDlg::OnPacket @0x6f56ea: `a1 = CInPacket::Decode1(a1)` mode read,
//	  then a switch with explicit cases 8/23/24/25/26/27 and a default arm
//	  that calls CParcelDlg::NoticeResult @0x6f5be2 for everything else
//	  (9-22, 28) and additionally triggers CloseParcelDlg when a1==18.
//	CParcelDlg::NoticeResult @0x6f5be2: two nested switches (a1<=16, a1>16)
//	  each mapping a mode value to one StringPool id; modes 9 and 20 hit no
//	  case in either switch and return without a notice (bodyless, silent).
//	PARCEL::Decode @0x4e4345: CInPacket::DecodeBuffer(234) fixed block, then
//	  Decode1(hasItem) and an optional GW_ItemSlotBase item.
//	CTabReceive::SetParcel @0x6ef69c (via CParcelDlg::SetParcelDlg
//	  @0x6f51f4, called from case 8): Decode1(mailbox count) + count *
//	  PARCEL::Decode, then Decode1(arrived count) + count * PARCEL::Decode.
//
// Full per-arm addresses and read orders are recorded in the spliced export
// entries (docs/packets/ida-exports/gms_v83.json, keys
// "CParcelDlg::OnPacket#<Arm>") and in the generated audit reports
// (docs/packets/audits/gms_v83/Parcel*.json). All 25 report verdicts are
// VerdictMatch (0) with FlatInvalid=false.

// packet-audit:verify packet=parcel/clientbound/ParcelSendEnableActions version=gms_v83 ida=0x6f5bf1
// packet-audit:verify packet=parcel/clientbound/ParcelNotEnoughMesos version=gms_v83 ida=0x6f5c83
// packet-audit:verify packet=parcel/clientbound/ParcelIncorrectRequest version=gms_v83 ida=0x6f5c6d
// packet-audit:verify packet=parcel/clientbound/ParcelNameDoesNotExist version=gms_v83 ida=0x6f5c57
// packet-audit:verify packet=parcel/clientbound/ParcelSameAccount version=gms_v83 ida=0x6f5c41
// packet-audit:verify packet=parcel/clientbound/ParcelReceiverStorageFull version=gms_v83 ida=0x6f5c2b
// packet-audit:verify packet=parcel/clientbound/ParcelReceiverUnableToReceive version=gms_v83 ida=0x6f5c15
// packet-audit:verify packet=parcel/clientbound/ParcelSenderUniqueConflict version=gms_v83 ida=0x6f5c99
// packet-audit:verify packet=parcel/clientbound/ParcelMesoLimit version=gms_v83 ida=0x6f5d11
// packet-audit:verify packet=parcel/clientbound/ParcelSuccessfullySent version=gms_v83 ida=0x6f579d
// packet-audit:verify packet=parcel/clientbound/ParcelUnknownError version=gms_v83 ida=0x6f5cb3
// packet-audit:verify packet=parcel/clientbound/ParcelRecvEnableActions version=gms_v83 ida=0x6f5cad
// packet-audit:verify packet=parcel/clientbound/ParcelRecvNoFreeSlots version=gms_v83 ida=0x6f5cd8
// packet-audit:verify packet=parcel/clientbound/ParcelRecvUniqueConflict version=gms_v83 ida=0x6f5cc5
// packet-audit:verify packet=parcel/clientbound/ParcelUnknownError2 version=gms_v83 ida=0x6f5ce9
func TestParcelResultArmsV83(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)

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

// packet-audit:verify packet=parcel/clientbound/ParcelOpenQuick version=gms_v83 ida=0x6f5862
func TestParcelOpenQuickV83(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 83, 1)
	got := NewParcelOpenQuick(0x1A).Encode(l, ctx)(nil)
	want := []byte{0x1A}
	if !bytes.Equal(got, want) {
		t.Errorf("OpenQuick: got % x want % x", got, want)
	}
}

// packet-audit:verify packet=parcel/clientbound/ParcelParcelRemoved version=gms_v83 ida=0x6f59fd
func TestParcelRemovedV83(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 83, 1)
	got := NewParcelRemoved(0x17, 7, ParcelRemovedKindDiscarded).Encode(l, ctx)(nil)
	want := []byte{0x17, 0x07, 0x00, 0x00, 0x00, 0x03}
	if !bytes.Equal(got, want) {
		t.Errorf("ParcelRemoved: got % x want % x", got, want)
	}
}

// packet-audit:verify packet=parcel/clientbound/ParcelParcelArrived version=gms_v83 ida=0x6f5997
func TestParcelArrivedV83(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 83, 1)
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

// TestParcelArrivedV83WithItem is a RULING 22 retro-fit (task-241 Task 28):
// the embedded Parcel's asset branch (parcel.go:157-173, HasItem/WriteBool +
// conditional item.Encode) is otherwise unexercised on every version, since
// every existing marked fixture builds a bare Parcel with no item attached.
// The parcel family itself carries no version gates
// (grep -rn 'MajorAtLeast|MajorVersion' libs/atlas-packet/parcel/ is empty);
// the only version-divergent bytes here belong to model.Asset, asserted
// independently per version via wantEquipItemBytesV83.
func TestParcelArrivedV83WithItem(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 83, 1)
	sentAt := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)
	item := model.NewAsset(false, 1, 1302000, time.Time{})
	p := parcel.NewParcel(7, "Alice", 1000, sentAt, "hi").SetItem(item)

	name := make([]byte, 13)
	copy(name, "Alice")
	msg := make([]byte, 205)
	copy(msg, "hi")
	filetime := model.MsTimeBytes(sentAt)

	var pBytes []byte
	pBytes = append(pBytes, 0x07, 0x00, 0x00, 0x00)
	pBytes = append(pBytes, name...)
	pBytes = append(pBytes, 0xe8, 0x03, 0x00, 0x00)
	pBytes = append(pBytes, filetime[:]...)
	pBytes = append(pBytes, msg...)
	pBytes = append(pBytes, 0x01) // hasItem = true
	pBytes = append(pBytes, wantEquipItemBytesV83()...)

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

// packet-audit:verify packet=parcel/clientbound/ParcelAlarmNamed version=gms_v83 ida=0x6f58b3
func TestParcelAlarmNamedV83(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 83, 1)
	got := NewParcelAlarmNamed(0x19, "Alice", true).Encode(l, ctx)(nil)
	var want []byte
	want = append(want, 0x19, 0x05, 0x00)
	want = append(want, []byte("Alice")...)
	want = append(want, 0x01)
	if !bytes.Equal(got, want) {
		t.Errorf("AlarmNamed: got % x want % x", got, want)
	}
}

// packet-audit:verify packet=parcel/clientbound/ParcelAlarmGeneric version=gms_v83 ida=0x6f580f
func TestParcelAlarmGenericV83(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 83, 1)
	got := NewParcelAlarmGeneric(0x1B, true).Encode(l, ctx)(nil)
	want := []byte{0x1B, 0x01}
	if !bytes.Equal(got, want) {
		t.Errorf("AlarmGeneric: got % x want % x", got, want)
	}
}

// packet-audit:verify packet=parcel/clientbound/ParcelOpen version=gms_v83 ida=0x6f5b32
func TestParcelOpenV83(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 83, 1)
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

// TestParcelOpenV83WithItem is a RULING 22 retro-fit companion to
// TestParcelArrivedV83WithItem: it exercises the same asset-bearing Parcel
// through the OPEN arm's mailbox/arrived slices.
func TestParcelOpenV83WithItem(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 83, 1)
	sentAt := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)
	item := model.NewAsset(false, 1, 1302000, time.Time{})
	p := parcel.NewParcel(7, "Alice", 1000, sentAt, "hi").SetItem(item)

	name := make([]byte, 13)
	copy(name, "Alice")
	msg := make([]byte, 205)
	copy(msg, "hi")
	filetime := model.MsTimeBytes(sentAt)

	var pBytes []byte
	pBytes = append(pBytes, 0x07, 0x00, 0x00, 0x00)
	pBytes = append(pBytes, name...)
	pBytes = append(pBytes, 0xe8, 0x03, 0x00, 0x00)
	pBytes = append(pBytes, filetime[:]...)
	pBytes = append(pBytes, msg...)
	pBytes = append(pBytes, 0x01) // hasItem = true
	pBytes = append(pBytes, wantEquipItemBytesV83()...)

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
