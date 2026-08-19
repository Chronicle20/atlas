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

// wantEquipItemBytesV84 hand-builds the wire encoding of a bare equipment
// asset (model.NewAsset(false, 1, 1302000, time.Time{})) under GMS v84, per
// model.Asset.encodeEquipableInfo (libs/atlas-packet/model/asset.go). v84 is
// on the far side of the MajorAtLeast(83) short-slot gate (asset.go:507) and
// the MajorAtLeast(79) hammersApplied gate (asset.go:266), same as v83 — but
// it ALSO crosses the MajorAtLeast(84) nDurability gate (asset.go:260-262),
// which v83 does not: an extra WriteInt32(-1) is inserted between experience
// and hammersApplied. Confirmed directly on this IDB (session 46c2a2eb):
// GW_ItemSlotEquip::RawDecode @0x4eaf34 issues three consecutive Decode4
// calls at 0x4eb134/0x4eb14e/0x4eb168, storing to this+200/this+212/this+224
// (experience/nDurability/hammersApplied) — one Decode4 more than the
// two-field v83 shape the asset.go comment cites. Every numeric field on
// this asset is its zero value, so each gated region below is either absent
// or all-zero-bytes of its fixed width; only the fixed byte offsets (and
// this one extra 4-byte field) differ from wantEquipItemBytesV83
// (task-241 Task 28, RULING 22).
func wantEquipItemBytesV84() []byte {
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
	b = append(b, 0xff, 0xff, 0xff, 0xff)                         // nDurability = -1 (asset.go:260-262, MajorAtLeast(84), ABSENT on v83)
	b = append(b, 0x00, 0x00, 0x00, 0x00)                         // hammersApplied = 0 (asset.go:266, MajorAtLeast(79))
	b = append(b, make([]byte, 8)...)                             // WriteLong(0) trailing buffer
	b = append(b, 0x00, 0x40, 0xe0, 0xfd, 0x3b, 0x37, 0x4f, 0x01) // 94354848000000000
	b = append(b, 0xff, 0xff, 0xff, 0xff)                         // -1
	return b
}

// v84 PARCEL (op 0x149) family verification (task-241 Task 28, batch 4/8,
// session 46c2a2eb, GMS_v84.1_U_DEVM.i64):
//
//	CParcelDlg::OnPacket_recv_0x149 @0x70db9f: `a1 = CInPacket::Decode1(a1)`
//	  mode read, then a switch with explicit cases 8/23/24/25/26/27 and a
//	  default arm that calls the NoticeResult equivalent (sub_70E097
//	  @0x70e097) for everything else (9-22, 28) and additionally triggers
//	  CloseParcelDlg (sub_70DB46) when a1==18 (@0x70dc52) — byte-identical
//	  switch shape to the gms_v83 anchor (RULING D; also independently
//	  confirmed by the pre-existing parcel.yaml provenance comment for this
//	  version, re-verified against THIS IDB this pass rather than leaned on).
//	sub_70E097 (NoticeResult equivalent) @0x70e097: two nested switches
//	  (a1<=16 @0x70e0a6, a1>16 @0x70e162) mapping mode values to StringPool
//	  ids; explicit cases 10/11/12/13/14/16 and default-a1==15 in the first
//	  switch, explicit cases 17/18/21/22 and 19/28-shared (LABEL_25) in the
//	  second. Modes 9 and 20 hit no case in either switch and fall through
//	  the "result = a1-15"/"result = a1-28" default with no notice call
//	  (bodyless, silent) — matches the v83-documented pattern. None of these
//	  9-22,28 arms consume any extra CInPacket bytes beyond the mode byte in
//	  OnPacket's default arm (@0x70dc07), so every notice-only arm's wire
//	  shape is just the single mode byte.
//	sub_4EB65F @0x4eb65f (PARCEL::Decode equivalent): CInPacket::DecodeBuffer
//	  (0xEA=234) fixed block, then Decode1(hasItem) and an optional
//	  GW_ItemSlotBase::Decode item — byte-identical shape to v83's
//	  PARCEL::Decode @0x4e4345.
//	sub_707B55 @0x707b55 (SetParcel equivalent, called from case 8 via
//	  sub_70D6DC @0x70d6dc): Decode1(mailbox count) + count * sub_4EB65F,
//	  then Decode1(arrived count) + count * sub_4EB65F.
//
// Full per-arm addresses and read orders are recorded in the spliced export
// entries (docs/packets/ida-exports/gms_v84.json, keys
// "CParcelDlg::OnPacket#<Arm>") and in the generated audit reports
// (docs/packets/audits/gms_v84/Parcel*.json).

// packet-audit:verify packet=parcel/clientbound/ParcelSendEnableActions version=gms_v84 ida=0x70e0a6
// packet-audit:verify packet=parcel/clientbound/ParcelNotEnoughMesos version=gms_v84 ida=0x70e138
// packet-audit:verify packet=parcel/clientbound/ParcelIncorrectRequest version=gms_v84 ida=0x70e122
// packet-audit:verify packet=parcel/clientbound/ParcelNameDoesNotExist version=gms_v84 ida=0x70e10c
// packet-audit:verify packet=parcel/clientbound/ParcelSameAccount version=gms_v84 ida=0x70e0f6
// packet-audit:verify packet=parcel/clientbound/ParcelReceiverStorageFull version=gms_v84 ida=0x70e0e0
// packet-audit:verify packet=parcel/clientbound/ParcelReceiverUnableToReceive version=gms_v84 ida=0x70e0ca
// packet-audit:verify packet=parcel/clientbound/ParcelSenderUniqueConflict version=gms_v84 ida=0x70e14e
// packet-audit:verify packet=parcel/clientbound/ParcelMesoLimit version=gms_v84 ida=0x70e1c6
// packet-audit:verify packet=parcel/clientbound/ParcelSuccessfullySent version=gms_v84 ida=0x70dc52
// packet-audit:verify packet=parcel/clientbound/ParcelUnknownError version=gms_v84 ida=0x70e19e
// packet-audit:verify packet=parcel/clientbound/ParcelRecvEnableActions version=gms_v84 ida=0x70e162
// packet-audit:verify packet=parcel/clientbound/ParcelRecvNoFreeSlots version=gms_v84 ida=0x70e18d
// packet-audit:verify packet=parcel/clientbound/ParcelRecvUniqueConflict version=gms_v84 ida=0x70e17a
// packet-audit:verify packet=parcel/clientbound/ParcelUnknownError2 version=gms_v84 ida=0x70e19e
func TestParcelResultArmsV84(t *testing.T) {
	ctx := pt.CreateContext("GMS", 84, 1)

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

// packet-audit:verify packet=parcel/clientbound/ParcelOpenQuick version=gms_v84 ida=0x70dd46
func TestParcelOpenQuickV84(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 84, 1)
	got := NewParcelOpenQuick(0x1A).Encode(l, ctx)(nil)
	want := []byte{0x1A}
	if !bytes.Equal(got, want) {
		t.Errorf("OpenQuick: got % x want % x", got, want)
	}
}

// packet-audit:verify packet=parcel/clientbound/ParcelParcelRemoved version=gms_v84 ida=0x70deb2
func TestParcelRemovedV84(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 84, 1)
	got := NewParcelRemoved(0x17, 7, ParcelRemovedKindDiscarded).Encode(l, ctx)(nil)
	want := []byte{0x17, 0x07, 0x00, 0x00, 0x00, 0x03}
	if !bytes.Equal(got, want) {
		t.Errorf("ParcelRemoved: got % x want % x", got, want)
	}
}

// packet-audit:verify packet=parcel/clientbound/ParcelParcelArrived version=gms_v84 ida=0x70de4c
func TestParcelArrivedV84(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 84, 1)
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

// TestParcelArrivedV84WithItem is a RULING 22 fixture (task-241 Task 28,
// batch 4/8): the embedded Parcel's asset branch (parcel.go:166-170,
// HasItem/WriteBool + conditional item.Encode) is the only version-divergent
// path in the whole PARCEL family (grep -rn 'MajorAtLeast|MajorVersion'
// libs/atlas-packet/parcel/ is empty) — and on v84 it is genuinely different
// from v83's bytes, not just re-derived for form: see wantEquipItemBytesV84.
func TestParcelArrivedV84WithItem(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 84, 1)
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
	pBytes = append(pBytes, wantEquipItemBytesV84()...)

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

// packet-audit:verify packet=parcel/clientbound/ParcelAlarmNamed version=gms_v84 ida=0x70dd68
func TestParcelAlarmNamedV84(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 84, 1)
	got := NewParcelAlarmNamed(0x19, "Alice", true).Encode(l, ctx)(nil)
	var want []byte
	want = append(want, 0x19, 0x05, 0x00)
	want = append(want, []byte("Alice")...)
	want = append(want, 0x01)
	if !bytes.Equal(got, want) {
		t.Errorf("AlarmNamed: got % x want % x", got, want)
	}
}

// packet-audit:verify packet=parcel/clientbound/ParcelAlarmGeneric version=gms_v84 ida=0x70dcc4
func TestParcelAlarmGenericV84(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 84, 1)
	got := NewParcelAlarmGeneric(0x1B, true).Encode(l, ctx)(nil)
	want := []byte{0x1B, 0x01}
	if !bytes.Equal(got, want) {
		t.Errorf("AlarmGeneric: got % x want % x", got, want)
	}
}

// packet-audit:verify packet=parcel/clientbound/ParcelOpen version=gms_v84 ida=0x70dfe7
func TestParcelOpenV84(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 84, 1)
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

// TestParcelOpenV84WithItem is a RULING 22 retro-fit companion to
// TestParcelArrivedV84WithItem: it exercises the same asset-bearing Parcel
// through the OPEN arm's mailbox/arrived slices (sub_707B55 @0x707b55).
func TestParcelOpenV84WithItem(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 84, 1)
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
	pBytes = append(pBytes, wantEquipItemBytesV84()...)

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
