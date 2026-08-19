package clientbound

import (
	"bytes"
	"testing"
	"time"

	testlog "github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-packet/parcel"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

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
