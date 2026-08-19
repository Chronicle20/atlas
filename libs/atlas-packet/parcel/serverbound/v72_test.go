package serverbound

import (
	"bytes"
	"testing"
)

// v72 DUEY_ACTION (op 0x040) verification — task-241 Task 28, session
// f2a2e7c1 (GMS_v72.1_U_DEVM.exe.i64). Every byte cites this IDB's own
// decompile directly:
//
//	CTabSend::SendParcel @0x65d940: NPC-path call site — COutPacket(&pkt,64)
//	  @0x65da57; Encode1(2) mode @0x65da65; Encode1(invType) @0x65da73;
//	  Encode2(slot) @0x65da82; Encode2(quantity) @0x65da91; Encode4(mesos)
//	  @0x65da9c; EncodeStr(recipientName) @0x65dab5; Encode1(0) @0x65dabe
//	  (quick=false) -- stops there, matching v83's asymmetric NPC-path
//	  shape. The quick-send counterpart is a SEPARATE named function in
//	  this IDB, CTabQuickSend::SendQuickDelivery @0x65c090 (unlike v83
//	  where it is an unnamed sub): same field/offset shape, Encode1(1)
//	  @0x65c2a0 (quick=true), then EncodeStr(message) @0x65c296 and
//	  Encode4(ticketRef) @0x65c2c4.
//	CTabReceive::ReceiveParcel @0x65af41: COutPacket(&pkt,64) @0x65afe4;
//	  Encode1(4) @0x65aff1; Encode4(parcelId) @0x65b004 -- gated on the
//	  30-day expiry countdown (parcel.go +21 field doc, already cites this
//	  same v72 address).
//	CTabReceive::DiscardParcel @0x65b061: COutPacket(&pkt,64) @0x65b0cb;
//	  Encode1(5) @0x65b0d8; Encode4(parcelId) @0x65b0eb.
//	CParcelDlg::CloseParcelDlg @0x65f8e1: COutPacket(&pkt,64) @0x65f8f6;
//	  Encode1(7) @0x65f904; no further bytes.
//
// The mode byte itself is written by the shared Action wrapper (action.go),
// not by these per-op structs, so it is not part of the byte sequences
// pinned below (mirrors the existing action_test.go discipline). Full
// per-op read orders are recorded in the spliced export entries
// (docs/packets/ida-exports/gms_v72.json) and in the generated audit
// reports (docs/packets/audits/gms_v72/ParcelAction*.json) — all four
// report verdicts are VerdictMatch (0) with FlatInvalid=false.

// packet-audit:verify packet=parcel/serverbound/ParcelActionSend version=gms_v72 ida=0x65d940
func TestActionSendV72(t *testing.T) {
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

// packet-audit:verify packet=parcel/serverbound/ParcelActionReceive version=gms_v72 ida=0x65af41
func TestActionReceiveV72(t *testing.T) {
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

// packet-audit:verify packet=parcel/serverbound/ParcelActionDiscard version=gms_v72 ida=0x65b061
func TestActionDiscardV72(t *testing.T) {
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

// packet-audit:verify packet=parcel/serverbound/ParcelActionClose version=gms_v72 ida=0x65f8e1
func TestActionCloseV72(t *testing.T) {
	l, ctx := newTestLogCtx()
	var a ActionClose
	got := a.Encode(l, ctx)(nil)
	if len(got) != 0 {
		t.Errorf("got % x want empty (mode written by the Action wrapper, not ActionClose)", got)
	}
}
