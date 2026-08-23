package serverbound

import (
	"bytes"
	"testing"
)

// jms_v185 DUEY_ACTION (op 0x039) verification (task-241 Task 28 batch 8/8,
// session a977912e, MapleStory_dump_SCY.exe.i64). Every byte cites this
// IDB's own decompile directly. Mode values are shifted +1 from every GMS
// build (docs/packets/dispatchers/duey_action.yaml):
//
//	CTabSend::SendParcel @0x753aa5: NPC path. COutPacket(&pkt,57) @0x753bbc;
//	  Encode1(3) mode @0x753bca; Encode1(invType=*(this+8)) @0x753bd8;
//	  Encode2(slot=*(this+12)) @0x753be7; Encode2(quantity=*(this+16))
//	  @0x753bf6; Encode4(mesos=*(this+20)) @0x753c01; EncodeStr(recipientName)
//	  @0x753c1a; Encode1(0) @0x753c23 quick=false -- stops there.
//	CTabQuickSend::SendQuickDelivery @0x75225f: quick path (named directly in
//	  this IDB). COutPacket(&pkt,57) @0x752407; Encode1(3) mode @0x752415;
//	  Encode1(invType=*(this+8)) @0x752423; Encode2(slot=*(this+12))
//	  @0x752432; Encode2(quantity=*(this+20)) @0x752441; Encode4(mesos=
//	  *(this+24)) @0x75244c; EncodeStr(recipientName) @0x752465; Encode1(1)
//	  @0x75246f quick=true; EncodeStr(message) @0x752488; Encode4(ticketRef=
//	  *(this+16)) @0x752493.
//	CTabReceive::ReceiveParcel @0x75110b: gated on the 30-day expiry
//	  countdown @0x751196; COutPacket(&pkt,0x39) @0x7511ae; Encode1(5)
//	  @0x7511bb; Encode4(parcelId) @0x7511ce.
//	CTabReceive::DiscardParcel @0x75122b: COutPacket(&pkt,57) @0x751295;
//	  Encode1(6) @0x7512a2; Encode4(parcelId) @0x7512b5.
//	CParcelDlg::CloseParcelDlg @0x7559c8: COutPacket(&pkt,57) @0x7559dd;
//	  Encode1(8) @0x7559eb; no further bytes.
//
// COutPacket::COutPacket(&pkt, 0x39/57) confirms opcode 57/0x039 at every
// call site, matching status.json's recorded jms_v185 opcode 57 and
// duey_action.yaml's own provenance. Body-matches every GMS build's shape
// one mode value later throughout.
//
// The mode byte itself is written by the shared Action wrapper (action.go),
// not by these per-op structs, so it is not part of the byte sequences
// pinned below (mirrors the existing action_test.go discipline). Full
// per-op read orders are recorded in the spliced export entries
// (docs/packets/ida-exports/gms_jms_185.json) and in the generated audit
// reports (docs/packets/audits/jms_v185/ParcelAction*.json) -- all four
// report verdicts are VerdictMatch (0) with FlatInvalid=false.

// packet-audit:verify packet=parcel/serverbound/ParcelActionSend version=jms_v185 ida=0x753aa5
func TestActionSendV185(t *testing.T) {
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

// packet-audit:verify packet=parcel/serverbound/ParcelActionReceive version=jms_v185 ida=0x75110b
func TestActionReceiveV185(t *testing.T) {
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

// packet-audit:verify packet=parcel/serverbound/ParcelActionDiscard version=jms_v185 ida=0x75122b
func TestActionDiscardV185(t *testing.T) {
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

// packet-audit:verify packet=parcel/serverbound/ParcelActionClose version=jms_v185 ida=0x7559c8
func TestActionCloseV185(t *testing.T) {
	l, ctx := newTestLogCtx()
	var a ActionClose
	got := a.Encode(l, ctx)(nil)
	if len(got) != 0 {
		t.Errorf("got % x want empty (mode written by the Action wrapper, not ActionClose)", got)
	}
}
