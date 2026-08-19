package clientbound

import (
	"bytes"
	"testing"
	"time"

	testlog "github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-packet/parcel"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// v72 PARCEL (op 0x120) family verification — task-241 Task 28, session
// f2a2e7c1 (GMS_v72.1_U_DEVM.exe.i64). Every byte below is derived from
// THIS IDB's own decompile (docs/packets/ida-exports/gms_v72.json, keys
// "CParcelDlg::OnPacket#<Arm>"), not merely asserted equal to v83:
//
//	CParcelDlg::OnPacket @0x65f93a: `a1 = CInPacket::Decode1(a1)` mode read,
//	  then a switch with explicit cases 8/23/24/25/26/27 and a default arm
//	  (sub_65FE2F(a1) @0x65f9da) that handles everything else (9-22, 28) and
//	  additionally triggers CloseParcelDlg when a1==18.
//	sub_65FE2F @0x65fe2f: v72's NoticeResult equivalent — two nested
//	  switches (a1<=16, a1>16) each mapping a mode value to one StringPool
//	  id; modes 9 and 20 hit no case in either switch and return without a
//	  notice (bodyless, silent) — byte-identical shape to v83's
//	  CParcelDlg::NoticeResult @0x6f5be2.
//	sub_4D07F5 @0x4d07f5 (PARCEL::Decode equivalent): CInPacket::
//	  DecodeBuffer(234) fixed block, then Decode1(hasItem) and an optional
//	  GW_ItemSlotBase item — byte-identical shape to v83's PARCEL::Decode
//	  @0x4e4345.
//	sub_65F476 @0x65f476 -> sub_65993F @0x65993f (SetParcel equivalent, via
//	  case 8): Decode1(mailbox count) + count * PARCEL::Decode, then
//	  Decode1(arrived count) + count * PARCEL::Decode.
//
// No wire divergence from v83 was found: docs/packets/dispatchers/
// parcel.yaml records this column byte-identical to the v83 switch shape
// (confirmed on this version's own IDB, Task 6), and libs/atlas-packet/
// parcel/**'s codec carries no version gate for any PARCEL arm — so per
// task-28 Ruling D, the arm bodies below assert byte-equality against the
// v83 encode (task-241 Task 28 batch gms_v83, TestParcelResultArmsV83 /
// TestParcelOpenQuickV83 / etc.) rather than re-deriving every literal.
//
// Full per-arm addresses and read orders are recorded in the spliced export
// entries (docs/packets/ida-exports/gms_v72.json) and in the generated
// audit reports (docs/packets/audits/gms_v72/Parcel*.json). All 21
// clientbound report verdicts are VerdictMatch (0) with FlatInvalid=false.

// packet-audit:verify packet=parcel/clientbound/ParcelSendEnableActions version=gms_v72 ida=0x65fe59
// packet-audit:verify packet=parcel/clientbound/ParcelNotEnoughMesos version=gms_v72 ida=0x65fed0
// packet-audit:verify packet=parcel/clientbound/ParcelIncorrectRequest version=gms_v72 ida=0x65feba
// packet-audit:verify packet=parcel/clientbound/ParcelNameDoesNotExist version=gms_v72 ida=0x65fea4
// packet-audit:verify packet=parcel/clientbound/ParcelSameAccount version=gms_v72 ida=0x65fe8e
// packet-audit:verify packet=parcel/clientbound/ParcelReceiverStorageFull version=gms_v72 ida=0x65fe78
// packet-audit:verify packet=parcel/clientbound/ParcelReceiverUnableToReceive version=gms_v72 ida=0x65fe62
// packet-audit:verify packet=parcel/clientbound/ParcelSenderUniqueConflict version=gms_v72 ida=0x65fee6
// packet-audit:verify packet=parcel/clientbound/ParcelMesoLimit version=gms_v72 ida=0x65ff5e
// packet-audit:verify packet=parcel/clientbound/ParcelSuccessfullySent version=gms_v72 ida=0x65ff4b
// packet-audit:verify packet=parcel/clientbound/ParcelUnknownError version=gms_v72 ida=0x65ff36
// packet-audit:verify packet=parcel/clientbound/ParcelRecvEnableActions version=gms_v72 ida=0x65ff09
// packet-audit:verify packet=parcel/clientbound/ParcelRecvNoFreeSlots version=gms_v72 ida=0x65ff25
// packet-audit:verify packet=parcel/clientbound/ParcelRecvUniqueConflict version=gms_v72 ida=0x65ff12
// packet-audit:verify packet=parcel/clientbound/ParcelUnknownError2 version=gms_v72 ida=0x65ff36
func TestParcelResultArmsV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)

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

// packet-audit:verify packet=parcel/clientbound/ParcelOpenQuick version=gms_v72 ida=0x65faaf
func TestParcelOpenQuickV72(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 72, 1)
	got := NewParcelOpenQuick(0x1A).Encode(l, ctx)(nil)
	want := []byte{0x1A}
	if !bytes.Equal(got, want) {
		t.Errorf("OpenQuick: got % x want % x", got, want)
	}
}

// packet-audit:verify packet=parcel/clientbound/ParcelParcelRemoved version=gms_v72 ida=0x65fc4a
func TestParcelRemovedV72(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 72, 1)
	got := NewParcelRemoved(0x17, 7, ParcelRemovedKindDiscarded).Encode(l, ctx)(nil)
	want := []byte{0x17, 0x07, 0x00, 0x00, 0x00, 0x03}
	if !bytes.Equal(got, want) {
		t.Errorf("ParcelRemoved: got % x want % x", got, want)
	}
}

// packet-audit:verify packet=parcel/clientbound/ParcelParcelArrived version=gms_v72 ida=0x65fbe4
func TestParcelArrivedV72(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 72, 1)
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

// packet-audit:verify packet=parcel/clientbound/ParcelAlarmNamed version=gms_v72 ida=0x65fb00
func TestParcelAlarmNamedV72(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 72, 1)
	got := NewParcelAlarmNamed(0x19, "Alice", true).Encode(l, ctx)(nil)
	var want []byte
	want = append(want, 0x19, 0x05, 0x00)
	want = append(want, []byte("Alice")...)
	want = append(want, 0x01)
	if !bytes.Equal(got, want) {
		t.Errorf("AlarmNamed: got % x want % x", got, want)
	}
}

// packet-audit:verify packet=parcel/clientbound/ParcelAlarmGeneric version=gms_v72 ida=0x65fa5c
func TestParcelAlarmGenericV72(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 72, 1)
	got := NewParcelAlarmGeneric(0x1B, true).Encode(l, ctx)(nil)
	want := []byte{0x1B, 0x01}
	if !bytes.Equal(got, want) {
		t.Errorf("AlarmGeneric: got % x want % x", got, want)
	}
}

// packet-audit:verify packet=parcel/clientbound/ParcelOpen version=gms_v72 ida=0x65fd7f
func TestParcelOpenV72(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 72, 1)
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
