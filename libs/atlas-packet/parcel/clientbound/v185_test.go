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

// wantEquipItemBytesV185 hand-builds the wire encoding of a bare equipment
// asset (model.NewAsset(true, 0, 1302000, time.Time{})) under JMS v185, per
// model.Asset.encodeEquipableInfo (libs/atlas-packet/model/asset.go).
//
// Independently decompiled GW_ItemSlotEquip::RawDecode @0x50feb9 (and its
// GW_ItemSlotBase::RawDecode base call @0x50f813) on session a977912e
// (MapleStory_dump_SCY.exe.i64), per Amendment 2's instruction to inherit
// neither the GMS v92+ 12-byte potential-trailer shape (RULING 24, commit
// a8adafb12) nor the pre-existing JMS-branch shape without confirming it
// here. Full read order after GW_ItemSlotBase::RawDecode (templateId(4) +
// isCash bool(1) + [cashId(8) if cash] + expiration DecodeBuffer(8)):
// Decode1(slots)+Decode1(level)+Decode1(the third Decode1 -- a JMS-only
// extra byte with no GMS counterpart, matching asset.go:240-242's
// `if t.Region()=="JMS" { WriteByte(0) }` gate)+15x Decode2 (the stat
// shorts, matching encodeEquipmentStats)+DecodeStr(owner name)+
// Decode2(flag)+Decode1(levelType)+Decode1(level, again)+Decode4(experience)
// +Decode4(hammersApplied) -- NOT the GMS v92+ nGrade/nCHUC/nOption1-3/
// nSocket1-2 byte+byte+short*5 (12-byte) block; instead exactly the
// pre-existing JMS-branch shape at asset.go:286-293 (byte+short*5+int,
// 15 bytes, all confirmed zero-value here) -- then an 8-byte buffer (skipped
// entirely when liCashItemSN is set, read when not), an unconditional
// second 8-byte buffer, and a trailing int32. This CONFIRMS the existing
// JMS branch in asset.go is correct for jms_v185: no wire divergence, no
// codec change needed (task-241 Task 28 batch 8/8).
func wantEquipItemBytesV185() []byte {
	var b []byte
	b = append(b, 0x01)                                                       // cash-type-byte marker (JMS always writes 1, asset.go:232-234)
	b = append(b, 0xf0, 0xdd, 0x13, 0x00)                                     // templateId 1302000 LE
	b = append(b, 0x00)                                                       // WriteBool(false) -- not cash
	b = append(b, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff)             // MsTime(zero) = -1
	b = append(b, 0x00)                                                       // slots = 0
	b = append(b, 0x00)                                                       // level = 0
	b = append(b, 0x00)                                                       // JMS-only extra byte (asset.go:240-242; 3rd Decode1 @0x50fefb)
	b = append(b, make([]byte, 30)...)                                        // 15 zero equipment stat shorts (15x Decode2 @0x50ff07-0x50ff5a)
	b = append(b, 0x00, 0x00)                                                 // WriteAsciiString("") -> short length 0
	b = append(b, 0x00, 0x00)                                                 // flag short = 0 (Decode2 @0x51005b area)
	b = append(b, 0x00)                                                       // levelType = 0
	b = append(b, 0x00)                                                       // level = 0 (again)
	b = append(b, 0x00, 0x00, 0x00, 0x00)                                     // experience = 0 (Decode4 @0x5100c7)
	b = append(b, 0x00, 0x00, 0x00, 0x00)                                     // hammersApplied = 0 (Decode4 @0x5100e1)
	b = append(b, 0x00)                                                       // JMS trailer byte (Decode1 @0x510106, asset.go:287)
	b = append(b, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00) // JMS trailer 5x short = 0 (Decode2 x5 @0x510115-0x51017d, asset.go:288-292)
	b = append(b, 0x00, 0x00, 0x00, 0x00)                                     // JMS trailer int = 0 (Decode4 @0x510197, asset.go:293)
	b = append(b, make([]byte, 8)...)                                         // WriteLong(0) trailing buffer (conditional-on-non-cash DecodeBuffer(8) @0x5101c2)
	b = append(b, 0x00, 0x40, 0xe0, 0xfd, 0x3b, 0x37, 0x4f, 0x01)             // 94354848000000000 (unconditional DecodeBuffer(8) @0x5101e2)
	b = append(b, 0xff, 0xff, 0xff, 0xff)                                     // -1 (Decode4 @0x5101ef)
	return b
}

// jms_v185 PARCEL (op 0x160) family verification (task-241 Task 28 batch
// 8/8, session a977912e, MapleStory_dump_SCY.exe.i64). CParcelDlg::OnPacket
// @0x755a21: `iPacket = CInPacket::Decode1(iPacket)` mode read @0x755a43,
// then an if/else-if chain over explicit cases 9/10/24/25/26/27/28 and a
// default arm that calls sub_755FD3 (the notice dispatcher) for everything
// else, additionally triggering CParcelDlg::CloseParcelDlg @0x7559c8 when
// iPacket==19 (checked @0x755ad9).
//
// jms_v185 is NOT byte-identical to v83/v84/v87/v92/v95 -- the mode space is
// shifted but NOT by a uniform delta (docs/packets/dispatchers/parcel.yaml
// CEILING note, confirmed independently here): OPEN body-matches at case 10
// (v83's case 8 + 1) -- case 9 is a genuinely NEW JMS-only arm (an outbound
// disconnect-time ack, COutPacket(&pkt,57)/Encode1(1)/Encode4(0)/
// Encode4(abs(v26)), with no v83 counterpart) -- while PARCEL_REMOVED/
// PARCEL_ARRIVED/ALARM_NAMED/OPEN_QUICK/ALARM_GENERIC body-match one-for-one
// at 24/25/26/27/28 (v83's 23/24/25/26/27 + 1), and SUCCESSFULLY_SENT
// body-matches at 19 (v83's 18 + 1, confirmed via the same
// CloseParcelDlg-on-match side effect in OnPacket's default arm).
// sub_755FD3 (JMS's NoticeResult dispatcher, @0x755fd3) drives the
// remaining 14 notice-only modes (12-23 minus the six explicit cases, plus
// 20/29) through StringPool ids with no wire body; per the CEILING note
// these cannot be mapped one-for-one to v83's operation keys without
// guessing (DISPATCHER_FAMILY.md), so parcel.yaml deliberately leaves them
// unset and this cell does not reach `verified` -- see status.json/
// STATUS.md and the batch report for the confirmed partial-coverage
// end state.
//
// Full per-arm addresses and read orders are recorded in the spliced export
// entries (docs/packets/ida-exports/gms_jms_185.json, keys
// "CParcelDlg::OnPacket#<Arm>") and in the generated audit reports
// (docs/packets/audits/jms_v185/Parcel*.json). All 7 claimable-arm report
// verdicts are VerdictMatch (0) with FlatInvalid=false.

// packet-audit:verify packet=parcel/clientbound/ParcelOpen version=jms_v185 ida=0x755e7b
func TestParcelOpenV185(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("JMS", 185, 1)
	sentAt := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)
	p := parcel.NewParcel(7, "Alice", 1000, sentAt, "hi")
	pBytes := p.Encode(l, ctx)(nil)

	got := NewParcelOpen(10, true, []parcel.Parcel{p}, []parcel.Parcel{p}).Encode(l, ctx)(nil)
	var want []byte
	want = append(want, 0x0a, 0x01, 0x01)
	want = append(want, pBytes...)
	want = append(want, 0x01)
	want = append(want, pBytes...)
	if !bytes.Equal(got, want) {
		t.Errorf("Open: got % x want % x", got, want)
	}
}

// TestParcelOpenV185WithItem is a RULING 22 fixture (task-241 Task 28): the
// embedded Parcel's asset branch (parcel.go:166-170, HasItem/WriteBool +
// conditional item.Encode) is otherwise unexercised on every version, since
// every other marked fixture builds a bare Parcel with no item attached. The
// parcel family itself carries no version gates
// (grep -rn MajorAtLeast libs/atlas-packet/parcel/ is empty); the only
// version-divergent bytes here belong to model.Asset, independently derived
// for JMS via wantEquipItemBytesV185 (Amendment 2 re-confirmation against
// THIS IDB's own GW_ItemSlotEquip::RawDecode, not inherited from any GMS
// fixture or the pre-existing JMS branch on faith).
func TestParcelOpenV185WithItem(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("JMS", 185, 1)
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
	pBytes = append(pBytes, wantEquipItemBytesV185()...)

	got := p.Encode(l, ctx)(nil)
	if !bytes.Equal(got, pBytes) {
		t.Fatalf("parcel with item: got % x\nwant % x", got, pBytes)
	}

	gotOpen := NewParcelOpen(10, true, []parcel.Parcel{p}, []parcel.Parcel{p}).Encode(l, ctx)(nil)
	var wantOpen []byte
	wantOpen = append(wantOpen, 0x0a, 0x01, 0x01)
	wantOpen = append(wantOpen, pBytes...)
	wantOpen = append(wantOpen, 0x01)
	wantOpen = append(wantOpen, pBytes...)
	if !bytes.Equal(gotOpen, wantOpen) {
		t.Errorf("Open with item: got % x want % x", gotOpen, wantOpen)
	}
}

// packet-audit:verify packet=parcel/clientbound/ParcelParcelRemoved version=jms_v185 ida=0x755d43
func TestParcelRemovedV185(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("JMS", 185, 1)
	got := NewParcelRemoved(24, 7, ParcelRemovedKindDiscarded).Encode(l, ctx)(nil)
	want := []byte{0x18, 0x07, 0x00, 0x00, 0x00, 0x03}
	if !bytes.Equal(got, want) {
		t.Errorf("ParcelRemoved: got % x want % x", got, want)
	}
}

// packet-audit:verify packet=parcel/clientbound/ParcelParcelArrived version=jms_v185 ida=0x755cdd
func TestParcelArrivedV185(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("JMS", 185, 1)
	sentAt := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)
	p := parcel.NewParcel(7, "Alice", 1000, sentAt, "hi")
	pBytes := p.Encode(l, ctx)(nil)

	got := NewParcelArrived(25, p).Encode(l, ctx)(nil)
	var want []byte
	want = append(want, 0x19)
	want = append(want, pBytes...)
	if !bytes.Equal(got, want) {
		t.Errorf("ParcelArrived: got % x want % x", got, want)
	}
}

// TestParcelArrivedV185WithItem is TestParcelOpenV185WithItem's companion
// for the single-parcel PARCEL_ARRIVED arm (RULING 22).
func TestParcelArrivedV185WithItem(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("JMS", 185, 1)
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
	pBytes = append(pBytes, wantEquipItemBytesV185()...)

	gotArrived := NewParcelArrived(25, p).Encode(l, ctx)(nil)
	var wantArrived []byte
	wantArrived = append(wantArrived, 0x19)
	wantArrived = append(wantArrived, pBytes...)
	if !bytes.Equal(gotArrived, wantArrived) {
		t.Errorf("ParcelArrived with item: got % x want % x", gotArrived, wantArrived)
	}
}

// packet-audit:verify packet=parcel/clientbound/ParcelAlarmNamed version=jms_v185 ida=0x755bf5
func TestParcelAlarmNamedV185(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("JMS", 185, 1)
	got := NewParcelAlarmNamed(26, "Alice", true).Encode(l, ctx)(nil)
	var want []byte
	want = append(want, 0x1a, 0x05, 0x00)
	want = append(want, []byte("Alice")...)
	want = append(want, 0x01)
	if !bytes.Equal(got, want) {
		t.Errorf("AlarmNamed: got % x want % x", got, want)
	}
}

// packet-audit:verify packet=parcel/clientbound/ParcelOpenQuick version=jms_v185 ida=0x755bcf
func TestParcelOpenQuickV185(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("JMS", 185, 1)
	got := NewParcelOpenQuick(27).Encode(l, ctx)(nil)
	want := []byte{0x1b}
	if !bytes.Equal(got, want) {
		t.Errorf("OpenQuick: got % x want % x", got, want)
	}
}

// packet-audit:verify packet=parcel/clientbound/ParcelAlarmGeneric version=jms_v185 ida=0x755b4b
func TestParcelAlarmGenericV185(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("JMS", 185, 1)
	got := NewParcelAlarmGeneric(28, true).Encode(l, ctx)(nil)
	want := []byte{0x1c, 0x01}
	if !bytes.Equal(got, want) {
		t.Errorf("AlarmGeneric: got % x want % x", got, want)
	}
}

// packet-audit:verify packet=parcel/clientbound/ParcelSuccessfullySent version=jms_v185 ida=0x755ad9
func TestParcelSuccessfullySentV185(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("JMS", 185, 1)
	got := NewParcelSuccessfullySent(19).Encode(l, ctx)(nil)
	want := []byte{0x13}
	if !bytes.Equal(got, want) {
		t.Errorf("SuccessfullySent: got % x want % x", got, want)
	}
}
