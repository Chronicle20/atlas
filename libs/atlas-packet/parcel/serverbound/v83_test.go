package serverbound

import (
	"bytes"
	"testing"
)

// v83 DUEY_ACTION (op 0x041) verification — gms_v83 IS THE ANCHOR (task-241
// Task 28, session 41f09cce, MapleStory_dump.exe.i64). Every byte cites this
// IDB's own decompile directly:
//
//	CTabSend::SendParcel: two independent client call sites share the op.
//	  NPC path (sub_6F36A8): COutPacket(&pkt,0x41); Encode1(2) mode;
//	  Encode1(invType)@0x6f37db; Encode2(slot)@0x6f37ea;
//	  Encode2(quantity)@0x6f37f9; Encode4(mesos)@0x6f3804;
//	  EncodeStr(recipientName)@0x6f381d; Encode1(0)@0x6f3826 quick=false —
//	  stops there. Quick path (sub_6F1DF5): same mode/invType/slot/quantity/
//	  mesos/recipientName shape at offsets +8/+12/+20/+24 (this-> layout
//	  differs slightly from the NPC struct), Encode1(1)@0x6f2005 quick=true,
//	  then EncodeStr(message)@0x6f201e + Encode4(ticketRef)@0x6f2029.
//	CTabReceive::ReceiveParcel @0x6f0ca3: COutPacket(&pkt,0x41);
//	  Encode1(4)@0x6f0d53; Encode4(parcelId)@0x6f0d66 — gated on the 30-day
//	  expiry countdown (parcel.go +21 field doc).
//	CTabReceive::DiscardParcel @0x6f0dc3: COutPacket(&pkt,65);
//	  Encode1(5)@0x6f0e3a; Encode4(parcelId)@0x6f0e4d.
//	CParcelDlg::CloseParcelDlg @0x6f5691: COutPacket(&pkt,65);
//	  Encode1(7)@0x6f56b4; no further bytes.
//
// The mode byte itself is written by the shared Action wrapper (action.go),
// not by these per-op structs, so it is not part of the byte sequences
// pinned below (mirrors the existing action_test.go discipline). Full
// per-op read orders are recorded in the spliced export entries
// (docs/packets/ida-exports/gms_v83.json) and in the generated audit
// reports (docs/packets/audits/gms_v83/ParcelAction*.json) — all four
// report verdicts are VerdictMatch (0) with FlatInvalid=false.

// packet-audit:verify packet=parcel/serverbound/ParcelActionSend version=gms_v83 ida=0x6f1df5
func TestActionSendV83(t *testing.T) {
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

// packet-audit:verify packet=parcel/serverbound/ParcelActionReceive version=gms_v83 ida=0x6f0ca3
func TestActionReceiveV83(t *testing.T) {
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

// packet-audit:verify packet=parcel/serverbound/ParcelActionDiscard version=gms_v83 ida=0x6f0dc3
func TestActionDiscardV83(t *testing.T) {
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

// packet-audit:verify packet=parcel/serverbound/ParcelActionClose version=gms_v83 ida=0x6f5691
func TestActionCloseV83(t *testing.T) {
	l, ctx := newTestLogCtx()
	var a ActionClose
	got := a.Encode(l, ctx)(nil)
	if len(got) != 0 {
		t.Errorf("got % x want empty (mode written by the Action wrapper, not ActionClose)", got)
	}
}
