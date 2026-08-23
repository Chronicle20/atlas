package serverbound

import (
	"bytes"
	"testing"
)

// v87 DUEY_ACTION (op 0x044) verification (task-241 Task 28, batch 5/8,
// session c0829805, GMSv87_4GB.exe.i64). Every byte cites this IDB's own
// decompile directly:
//
//	CTabSend::SendParcel: two independent client call sites share the op,
//	  same asymmetric shape as v83/v84.
//	  NPC path CTabSend::SendParcel @0x73269a: COutPacket(&pkt,0x44)
//	  @0x7327b1; Encode1(2)@0x7327bf mode; Encode1(invType)@0x7327cd;
//	  Encode2(slot)@0x7327dc; Encode2(quantity)@0x7327eb;
//	  Encode4(mesos)@0x7327f6; EncodeStr(recipientName)@0x73280f;
//	  Encode1(0)@0x732818 quick=false -- stops there.
//	  Quick path CTabQuickSend::SendQuickDelivery @0x730de5:
//	  COutPacket(&pkt,0x44)@0x730f8d; Encode1(2)@0x730f9b mode;
//	  Encode1(invType)@0x730fa9; Encode2(slot)@0x730fb8;
//	  Encode2(quantity)@0x730fc7; Encode4(mesos)@0x730fd2;
//	  EncodeStr(recipientName)@0x730feb; Encode1(1)@0x730ff5 quick=true;
//	  EncodeStr(message)@0x73100e; Encode4(ticketRef)@0x731019.
//	CTabReceive::ReceiveParcel @0x72fc91: COutPacket(&pkt,0x44)@0x72fd34;
//	  Encode1(4)@0x72fd41; Encode4(parcelId)@0x72fd54 -- gated on the
//	  30-day expiry countdown (same shape as v83/v84).
//	CTabReceive::DiscardParcel @0x72fdb1: COutPacket(&pkt,0x44)@0x72fe1b;
//	  Encode1(5)@0x72fe28; Encode4(parcelId)@0x72fe3b.
//	CParcelDlg::CloseParcelDlg @0x734682: COutPacket(&pkt,0x44)@0x734697;
//	  Encode1(7)@0x7346a5; no further bytes.
//
// The mode byte itself is written by the shared Action wrapper (action.go),
// not by these per-op structs, so it is not part of the byte sequences
// pinned below (mirrors the existing action_test.go/v83_test.go/v84_test.go
// discipline). Full per-op read orders are recorded in the spliced export
// entries (docs/packets/ida-exports/gms_v87.json) and in the generated
// audit reports (docs/packets/audits/gms_v87/ParcelAction*.json).

// packet-audit:verify packet=parcel/serverbound/ParcelActionSend version=gms_v87 ida=0x730de5
func TestActionSendV87(t *testing.T) {
	l, ctx := newTestLogCtx()

	t.Run("npc", func(t *testing.T) {
		buf := sendNpcBytes()
		r := newReader(buf)
		var a ActionSend
		a.Decode(l, ctx)(&r, nil)

		got := a.Encode(l, ctx)(nil)
		if !bytes.Equal(got, buf) {
			t.Errorf("npc round-trip: got % x want % x", got, buf)
		}
	})

	t.Run("quick", func(t *testing.T) {
		buf := sendQuickBytes()
		r := newReader(buf)
		var a ActionSend
		a.Decode(l, ctx)(&r, nil)

		got := a.Encode(l, ctx)(nil)
		if !bytes.Equal(got, buf) {
			t.Errorf("quick round-trip: got % x want % x", got, buf)
		}
	})
}

// packet-audit:verify packet=parcel/serverbound/ParcelActionReceive version=gms_v87 ida=0x72fc91
func TestActionReceiveV87(t *testing.T) {
	l, ctx := newTestLogCtx()
	buf := []byte{0x07, 0x00, 0x00, 0x00}
	r := newReader(buf)
	var a ActionReceive
	a.Decode(l, ctx)(&r, nil)

	got := a.Encode(l, ctx)(nil)
	if !bytes.Equal(got, buf) {
		t.Errorf("got % x want % x", got, buf)
	}
}

// packet-audit:verify packet=parcel/serverbound/ParcelActionDiscard version=gms_v87 ida=0x72fdb1
func TestActionDiscardV87(t *testing.T) {
	l, ctx := newTestLogCtx()
	buf := []byte{0x07, 0x00, 0x00, 0x00}
	r := newReader(buf)
	var a ActionDiscard
	a.Decode(l, ctx)(&r, nil)

	got := a.Encode(l, ctx)(nil)
	if !bytes.Equal(got, buf) {
		t.Errorf("got % x want % x", got, buf)
	}
}

// packet-audit:verify packet=parcel/serverbound/ParcelActionClose version=gms_v87 ida=0x734682
func TestActionCloseV87(t *testing.T) {
	l, ctx := newTestLogCtx()
	var a ActionClose
	got := a.Encode(l, ctx)(nil)
	if len(got) != 0 {
		t.Errorf("got % x want empty (mode written by the Action wrapper, not ActionClose)", got)
	}
}
