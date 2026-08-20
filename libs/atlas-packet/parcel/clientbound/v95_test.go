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

// wantEquipItemBytesV95 hand-builds the wire encoding of a bare equipment
// asset (model.NewAsset(false, 1, 1302000, time.Time{})) under GMS v95, per
// model.Asset.encodeEquipableInfo (libs/atlas-packet/model/asset.go).
//
// Re-decompiled GW_ItemSlotEquip::RawDecode @0x4f8360 on session ecc757f4
// (GMS_v95.0_U_DEVM.exe.i64), per RULING 24's instruction not to inherit the
// shape unconfirmed. Full read order after GW_ItemSlotBase::RawDecode:
// Decode1(nRUC)+Decode1(nCUC)+15x Decode2 (the stat shorts, matching
// model.Asset.encodeEquipmentStats at asset.go:530-545)+DecodeStr(sTitle)+
// Decode2(nAttribute)+Decode1(nLevelUpType)+Decode1(nLevel)+Decode4(nEXP)+
// Decode4(nDurability @0x4f8587)+Decode4(nIUC/hammersApplied @0x4f85a1)+
// Decode1(nGrade @0x4f85bb)+Decode1(nCHUC @0x4f85d4)+Decode2(nOption1
// @0x4f85e8)+Decode2(nOption2 @0x4f85fd)+Decode2(nOption3 @0x4f8612)+
// Decode2(nSocket1 @0x4f8627)+Decode2(nSocket2 @0x4f863c) -- the same
// nGrade/nCHUC/nOption1-3/nSocket1-2 12-byte potential block gated behind
// GMS MajorAtLeast(92) in asset.go, byte-identical in shape and position to
// v92's wantEquipItemBytesV92 (RULING 24 fix, commit a8adafb12). v95 crosses
// the same set of asset.go gates as v92 (MajorAtLeast(72)/(79)/(83)/(84)/(92)
// all true for v95), so every gated region below is either present or absent
// in the same pattern as v92; only the numeric field values (all zero on
// this bare fixture) differ in no way from v92's (task-241 Task 28 batch
// 7/8, RULING 22 retro-fit).
func wantEquipItemBytesV95() []byte {
	var b []byte
	b = append(b, 0x01, 0x00)                                                 // encodeSlot: short slot (MajorAtLeast(83) true, asset.go:523)
	b = append(b, 0x01)                                                       // cash-type-byte marker (MajorVersion>12)
	b = append(b, 0xf0, 0xdd, 0x13, 0x00)                                     // templateId 1302000 LE
	b = append(b, 0x00)                                                       // WriteBool(false) -- not cash
	b = append(b, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff)             // MsTime(zero) = -1
	b = append(b, 0x00)                                                       // slots = 0
	b = append(b, 0x00)                                                       // level = 0
	b = append(b, make([]byte, 30)...)                                        // 15 zero equipment stat shorts
	b = append(b, 0x00, 0x00)                                                 // WriteAsciiString("") -> short length 0
	b = append(b, 0x00, 0x00)                                                 // flag short = 0
	b = append(b, 0x00)                                                       // levelType = 0
	b = append(b, 0x00)                                                       // level = 0
	b = append(b, 0x00, 0x00, 0x00, 0x00)                                     // experience = 0
	b = append(b, 0xff, 0xff, 0xff, 0xff)                                     // nDurability = -1 (asset.go:260-262, MajorAtLeast(84))
	b = append(b, 0x00, 0x00, 0x00, 0x00)                                     // hammersApplied = 0 (asset.go:266, MajorAtLeast(79))
	b = append(b, 0x00, 0x00)                                                 // nGrade, nCHUC = 0, 0 (Decode1 @0x4f85bb/0x4f85d4, MajorAtLeast(92))
	b = append(b, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00) // nOption1-3, nSocket1-2 = 0 (5x Decode2 @0x4f85e8/0x4f85fd/0x4f8612/0x4f8627/0x4f863c, MajorAtLeast(92))
	b = append(b, make([]byte, 8)...)                                         // WriteLong(0) trailing buffer
	b = append(b, 0x00, 0x40, 0xe0, 0xfd, 0x3b, 0x37, 0x4f, 0x01)             // 94354848000000000
	b = append(b, 0xff, 0xff, 0xff, 0xff)                                     // -1
	return b
}

// v95 PARCEL (op 0x17D) family verification (task-241 Task 28 batch 7/8,
// session ecc757f4, GMS_v95.0_U_DEVM.exe.i64). CParcelDlg::OnPacket
// @0x692970: `v1 = CInPacket::Decode1(iPacket)` mode read @0x6929a8, then a
// switch with explicit cases 8/23/24/25/26/27 and a default arm that calls
// CParcelDlg::NoticeResult @0x68efd0 for everything else (9-22, 28), and
// additionally triggers CParcelDlg::CloseParcelDlg @0x68ef40 when v1==18
// (checked @0x692ec1) -- byte-identical switch shape to v83/v84/v87/v92,
// independently re-confirmed against THIS IDB (RULING D).
// CParcelDlg::NoticeResult @0x68efd0: a single switch(nResult) @0x68efe0
// with explicit cases 10/11/12/13/14/15/16/17/18/19(&28)/21/22; modes 9 and
// 20 hit no case and return without a notice (bodyless, silent -- same
// pattern as v83/v84/v87/v92). PARCEL::Decode @0x4f88a0:
// CInPacket::DecodeBuffer(0xEA=234) @0x4f88d3 fixed block, then
// Decode1(hasItem) @0x4f88da and an optional GW_ItemSlotBase::Decode
// @0x4f88ec item. CTabReceive::SetParcel @0x692560 (via
// CParcelDlg::SetParcelDlg @0x692960, called from case 8): Decode1(mailbox
// count) @0x692596 + count * PARCEL::Decode @0x6925cd (loop
// @0x6925ac-0x692667), then Decode1(arrived count) @0x6926e0 + count *
// PARCEL::Decode @0x692714 (loop @0x6926f2-0x692942).
//
// Full per-arm addresses and read orders are recorded in the spliced export
// entries (docs/packets/ida-exports/gms_v95.json, keys
// "CParcelDlg::OnPacket#<Arm>") and in the generated audit reports
// (docs/packets/audits/gms_v95/Parcel*.json). All 25 report verdicts are
// VerdictMatch (0) with FlatInvalid=false.

// packet-audit:verify packet=parcel/clientbound/ParcelSendEnableActions version=gms_v95 ida=0x68efe0
// packet-audit:verify packet=parcel/clientbound/ParcelNotEnoughMesos version=gms_v95 ida=0x68f001
// packet-audit:verify packet=parcel/clientbound/ParcelIncorrectRequest version=gms_v95 ida=0x68f01b
// packet-audit:verify packet=parcel/clientbound/ParcelNameDoesNotExist version=gms_v95 ida=0x68f034
// packet-audit:verify packet=parcel/clientbound/ParcelSameAccount version=gms_v95 ida=0x68f04e
// packet-audit:verify packet=parcel/clientbound/ParcelReceiverStorageFull version=gms_v95 ida=0x68f068
// packet-audit:verify packet=parcel/clientbound/ParcelReceiverUnableToReceive version=gms_v95 ida=0x68f081
// packet-audit:verify packet=parcel/clientbound/ParcelSenderUniqueConflict version=gms_v95 ida=0x68f098
// packet-audit:verify packet=parcel/clientbound/ParcelMesoLimit version=gms_v95 ida=0x68efe7
// packet-audit:verify packet=parcel/clientbound/ParcelSuccessfullySent version=gms_v95 ida=0x692ec1
// packet-audit:verify packet=parcel/clientbound/ParcelUnknownError version=gms_v95 ida=0x68f0c5
// packet-audit:verify packet=parcel/clientbound/ParcelRecvEnableActions version=gms_v95 ida=0x68efe0
// packet-audit:verify packet=parcel/clientbound/ParcelRecvNoFreeSlots version=gms_v95 ida=0x68f0dc
// packet-audit:verify packet=parcel/clientbound/ParcelRecvUniqueConflict version=gms_v95 ida=0x68f0f3
// packet-audit:verify packet=parcel/clientbound/ParcelUnknownError2 version=gms_v95 ida=0x68f0c5
func TestParcelResultArmsV95(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)

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

// packet-audit:verify packet=parcel/clientbound/ParcelOpenQuick version=gms_v95 ida=0x692a15
func TestParcelOpenQuickV95(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 95, 1)
	got := NewParcelOpenQuick(0x1A).Encode(l, ctx)(nil)
	want := []byte{0x1A}
	if !bytes.Equal(got, want) {
		t.Errorf("OpenQuick: got % x want % x", got, want)
	}
}

// packet-audit:verify packet=parcel/clientbound/ParcelParcelRemoved version=gms_v95 ida=0x692aca
func TestParcelRemovedV95(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 95, 1)
	got := NewParcelRemoved(0x17, 7, ParcelRemovedKindDiscarded).Encode(l, ctx)(nil)
	want := []byte{0x17, 0x07, 0x00, 0x00, 0x00, 0x03}
	if !bytes.Equal(got, want) {
		t.Errorf("ParcelRemoved: got % x want % x", got, want)
	}
}

// packet-audit:verify packet=parcel/clientbound/ParcelParcelArrived version=gms_v95 ida=0x692c46
func TestParcelArrivedV95(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 95, 1)
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

// TestParcelArrivedV95WithItem is a RULING 22 fixture (task-241 Task 28):
// the embedded Parcel's asset branch (parcel.go:166-170, HasItem/WriteBool +
// conditional item.Encode) is otherwise unexercised on every version, since
// every other marked fixture builds a bare Parcel with no item attached. The
// parcel family itself carries no version gates
// (grep -rn MajorAtLeast libs/atlas-packet/parcel/ is empty); the only
// version-divergent bytes here belong to model.Asset, asserted independently
// per version via wantEquipItemBytesV95 -- byte-identical in shape to
// wantEquipItemBytesV92 because v95 crosses the exact same set of asset.go
// gates as v92 (confirmed by decompiling GW_ItemSlotEquip::RawDecode on
// THIS IDB, not assumed from v92's fixture).
func TestParcelArrivedV95WithItem(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 95, 1)
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
	pBytes = append(pBytes, wantEquipItemBytesV95()...)

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

// packet-audit:verify packet=parcel/clientbound/ParcelAlarmNamed version=gms_v95 ida=0x692cb6
func TestParcelAlarmNamedV95(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 95, 1)
	got := NewParcelAlarmNamed(0x19, "Alice", true).Encode(l, ctx)(nil)
	var want []byte
	want = append(want, 0x19, 0x05, 0x00)
	want = append(want, []byte("Alice")...)
	want = append(want, 0x01)
	if !bytes.Equal(got, want) {
		t.Errorf("AlarmNamed: got % x want % x", got, want)
	}
}

// packet-audit:verify packet=parcel/clientbound/ParcelAlarmGeneric version=gms_v95 ida=0x692dde
func TestParcelAlarmGenericV95(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 95, 1)
	got := NewParcelAlarmGeneric(0x1B, true).Encode(l, ctx)(nil)
	want := []byte{0x1B, 0x01}
	if !bytes.Equal(got, want) {
		t.Errorf("AlarmGeneric: got % x want % x", got, want)
	}
}

// packet-audit:verify packet=parcel/clientbound/ParcelOpen version=gms_v95 ida=0x692a73
func TestParcelOpenV95(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 95, 1)
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

// TestParcelOpenV95WithItem is a RULING 22 companion to
// TestParcelArrivedV95WithItem: it exercises the same asset-bearing Parcel
// through the OPEN arm's mailbox/arrived slices.
func TestParcelOpenV95WithItem(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 95, 1)
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
	pBytes = append(pBytes, wantEquipItemBytesV95()...)

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
