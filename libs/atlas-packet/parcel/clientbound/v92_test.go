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

// wantEquipItemBytesV92 hand-builds the wire encoding of a bare equipment
// asset (model.NewAsset(true, 0, 1302000, time.Time{})) under GMS v92, per
// model.Asset.encodeEquipableInfo (libs/atlas-packet/model/asset.go).
//
// CORRECTED (task-241 Task 28 batch-6 fix round, bug
// docs/tasks/task-241-duey-parcel-delivery/bug-v92-equip-trailer-ungated.md):
// batch 6 asserted this fixture as byte-identical to v84/v87 from a
// three-Decode4 shape check that stopped before the actual divergence.
// Re-decompiled GW_ItemSlotEquip::RawDecode @0x4f35d0 on session 019cd393
// (GMS_v92_1_DEVM.exe.i64): after the three Decode4 reads (experience +233,
// nDurability +245, hammersApplied +257 -- confirmed same as v84/v87), v92
// reads SEVEN MORE fields that v84/v87 do not: Decode1 (+263), Decode1
// (+269), Decode2 (+277), Decode2 (+285), Decode2 (+293), Decode2 (+301),
// Decode2 (+309) -- 2 bytes + 5 shorts = 12 extra bytes (the standard
// nGrade/nCHUC + nOption1-3/nSocket1-2 potential block) -- before reaching
// the conditional DecodeBuffer(+40,8)/DecodeBuffer(+61,8)/Decode4(+69) that
// v84 (@0x4eaf34, re-confirmed) and v87 reach immediately after the third
// Decode4. Mirrored in model.Asset.encodeEquipableInfo/decodeEquipableInfo
// behind GMS MajorAtLeast(92) (asset.go). This is NOT the JMS branch (which
// writes 1 byte + 5 shorts + 1 int = 15 bytes) -- v92 has no trailing int.
func wantEquipItemBytesV92() []byte {
	var b []byte
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
	b = append(b, 0xff, 0xff, 0xff, 0xff)                                     // nDurability = -1 (asset.go:260-262, MajorAtLeast(84), ABSENT on v83)
	b = append(b, 0x00, 0x00, 0x00, 0x00)                                     // hammersApplied = 0 (asset.go:266, MajorAtLeast(79))
	b = append(b, 0x00, 0x00)                                                 // nGrade, nCHUC = 0, 0 (Decode1 @+263/+269, MajorAtLeast(92))
	b = append(b, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00) // nOption1-3, nSocket1-2 = 0 (5x Decode2 @+277/+285/+293/+301/+309, MajorAtLeast(92))
	b = append(b, make([]byte, 8)...)                                         // WriteLong(0) trailing buffer
	b = append(b, 0x00, 0x40, 0xe0, 0xfd, 0x3b, 0x37, 0x4f, 0x01)             // 94354848000000000
	b = append(b, 0xff, 0xff, 0xff, 0xff)                                     // -1
	return b
}

// v92 PARCEL (op 0x175) family verification (task-241 Task 28 batch 6/8,
// session 019cd393, GMS_v92_1_DEVM.exe.i64). CParcelDlg::OnPacket @0x689190:
// `v1 = CInPacket::Decode1(a1)` mode read @0x6891c8, then a switch with
// explicit cases 8/23/24/25/26/27 and a default arm that calls a
// NoticeResult-equivalent (sub_6807D0 @0x6807d0) for everything else
// (9-22, 28), and additionally triggers CParcelDlg::CloseParcelDlg @0x680740
// when v1==18 (checked @0x6896e1) -- byte-identical switch shape to
// v83/v84/v87, independently re-confirmed against THIS IDB (RULING D).
// sub_6807D0: a single switch(a1) @0x6807e0 with explicit cases
// 10/11/12/13/14/15/16/17/18/19/21/22/28 (19 and 28 share a body); modes 9
// and 20 hit no case and return without a notice (bodyless, silent -- same
// pattern as v83/v84/v87). PARCEL::Decode @0x4f3b10:
// CInPacket::DecodeBuffer(234) fixed block, then Decode1(hasItem) and an
// optional GW_ItemSlotBase item. CTabReceive::SetParcel @0x688da0 (via
// CParcelDlg::SetParcelDlg @0x689180, called from case 8): Decode1(mailbox
// count) + count * PARCEL::Decode, then Decode1(arrived count) + count *
// PARCEL::Decode.
//
// Full per-arm addresses and read orders are recorded in the spliced export
// entries (docs/packets/ida-exports/gms_v92.json, keys
// "CParcelDlg::OnPacket#<Arm>") and in the generated audit reports
// (docs/packets/audits/gms_v92/Parcel*.json). All 25 report verdicts are
// VerdictMatch (0) with FlatInvalid=false.

// packet-audit:verify packet=parcel/clientbound/ParcelSendEnableActions version=gms_v92 ida=0x6807e0
// packet-audit:verify packet=parcel/clientbound/ParcelNotEnoughMesos version=gms_v92 ida=0x680801
// packet-audit:verify packet=parcel/clientbound/ParcelIncorrectRequest version=gms_v92 ida=0x68081b
// packet-audit:verify packet=parcel/clientbound/ParcelNameDoesNotExist version=gms_v92 ida=0x680834
// packet-audit:verify packet=parcel/clientbound/ParcelSameAccount version=gms_v92 ida=0x68084e
// packet-audit:verify packet=parcel/clientbound/ParcelReceiverStorageFull version=gms_v92 ida=0x680868
// packet-audit:verify packet=parcel/clientbound/ParcelReceiverUnableToReceive version=gms_v92 ida=0x680881
// packet-audit:verify packet=parcel/clientbound/ParcelSenderUniqueConflict version=gms_v92 ida=0x680898
// packet-audit:verify packet=parcel/clientbound/ParcelMesoLimit version=gms_v92 ida=0x6807e7
// packet-audit:verify packet=parcel/clientbound/ParcelSuccessfullySent version=gms_v92 ida=0x6896e1
// packet-audit:verify packet=parcel/clientbound/ParcelUnknownError version=gms_v92 ida=0x6808c5
// packet-audit:verify packet=parcel/clientbound/ParcelRecvEnableActions version=gms_v92 ida=0x6807e0
// packet-audit:verify packet=parcel/clientbound/ParcelRecvNoFreeSlots version=gms_v92 ida=0x6808dc
// packet-audit:verify packet=parcel/clientbound/ParcelRecvUniqueConflict version=gms_v92 ida=0x6808f3
// packet-audit:verify packet=parcel/clientbound/ParcelUnknownError2 version=gms_v92 ida=0x6808c5
func TestParcelResultArmsV92(t *testing.T) {
	ctx := pt.CreateContext("GMS", 92, 1)

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

// packet-audit:verify packet=parcel/clientbound/ParcelOpenQuick version=gms_v92 ida=0x689235
func TestParcelOpenQuickV92(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 92, 1)
	got := NewParcelOpenQuick(0x1A).Encode(l, ctx)(nil)
	want := []byte{0x1A}
	if !bytes.Equal(got, want) {
		t.Errorf("OpenQuick: got % x want % x", got, want)
	}
}

// packet-audit:verify packet=parcel/clientbound/ParcelParcelRemoved version=gms_v92 ida=0x6892ea
func TestParcelRemovedV92(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 92, 1)
	got := NewParcelRemoved(0x17, 7, ParcelRemovedKindDiscarded).Encode(l, ctx)(nil)
	want := []byte{0x17, 0x07, 0x00, 0x00, 0x00, 0x03}
	if !bytes.Equal(got, want) {
		t.Errorf("ParcelRemoved: got % x want % x", got, want)
	}
}

// packet-audit:verify packet=parcel/clientbound/ParcelParcelArrived version=gms_v92 ida=0x689466
func TestParcelArrivedV92(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 92, 1)
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

// TestParcelArrivedV92WithItem is a RULING 22 fixture (task-241 Task 28):
// the embedded Parcel's asset branch (parcel.go:166-170, HasItem/WriteBool +
// conditional item.Encode) is otherwise unexercised on every version, since
// every other marked fixture builds a bare Parcel with no item attached. The
// parcel family itself carries no version gates
// (grep -rn MajorAtLeast libs/atlas-packet/parcel/ is empty); the only
// version-divergent bytes here belong to model.Asset, asserted independently
// per version via wantEquipItemBytesV92 (12 bytes longer than
// wantEquipItemBytesV87/V84 -- GMS v92+ equip trailer gate, see the doc
// comment on wantEquipItemBytesV92 above).
func TestParcelArrivedV92WithItem(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 92, 1)
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
	pBytes = append(pBytes, wantEquipItemBytesV92()...)

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

// packet-audit:verify packet=parcel/clientbound/ParcelAlarmNamed version=gms_v92 ida=0x6894d6
func TestParcelAlarmNamedV92(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 92, 1)
	got := NewParcelAlarmNamed(0x19, "Alice", true).Encode(l, ctx)(nil)
	var want []byte
	want = append(want, 0x19, 0x05, 0x00)
	want = append(want, []byte("Alice")...)
	want = append(want, 0x01)
	if !bytes.Equal(got, want) {
		t.Errorf("AlarmNamed: got % x want % x", got, want)
	}
}

// packet-audit:verify packet=parcel/clientbound/ParcelAlarmGeneric version=gms_v92 ida=0x6895fe
func TestParcelAlarmGenericV92(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 92, 1)
	got := NewParcelAlarmGeneric(0x1B, true).Encode(l, ctx)(nil)
	want := []byte{0x1B, 0x01}
	if !bytes.Equal(got, want) {
		t.Errorf("AlarmGeneric: got % x want % x", got, want)
	}
}

// packet-audit:verify packet=parcel/clientbound/ParcelOpen version=gms_v92 ida=0x689293
func TestParcelOpenV92(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 92, 1)
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

// TestParcelOpenV92WithItem is a RULING 22 companion to
// TestParcelArrivedV92WithItem: it exercises the same asset-bearing Parcel
// through the OPEN arm's mailbox/arrived slices.
func TestParcelOpenV92WithItem(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 92, 1)
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
	pBytes = append(pBytes, wantEquipItemBytesV92()...)

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
