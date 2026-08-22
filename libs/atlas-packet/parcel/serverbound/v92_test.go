package serverbound

import (
	"bytes"
	"testing"
)

// v92 DUEY_ACTION (op 0x047) verification (task-241 Task 28 batch 6/8,
// session 019cd393, GMS_v92_1_DEVM.exe.i64). Every byte cites this IDB's own
// decompile directly:
//
//	CTabSend::SendParcel @0x683c20: two independent client call sites share
//	  the op. NPC path (CTabSend::SendParcel itself): COutPacket(&pkt,0x47)
//	  @0x683d98; Encode1(2) mode @0x683da8; Encode1(invType) @0x683db6;
//	  Encode2(slot) @0x683dc4; Encode2(quantity) @0x683dd2; Encode4(mesos)
//	  @0x683ddf; EncodeStr(recipientName) @0x683dfb; Encode1(0) @0x683e05
//	  quick=false -- stops there. Quick path (sub_682DA0 @0x682da0, unnamed
//	  in this IDB -- located via the '6A 47 8D 4C 24' push-opcode/lea-ecx/
//	  call-COutPacket-ctor byte signature, task-092 "distrust IDB names"):
//	  same mode/invType/slot/quantity/mesos/recipientName shape at
//	  @0x682fe6/0x682ff4/0x683002/0x683010/0x68301d/0x683039 (this-> layout
//	  differs slightly from the NPC struct), Encode1(1) quick=true @0x683044,
//	  then EncodeStr(message) @0x683060 + Encode4(ticketRef) @0x68306d.
//	CTabReceive::ReceiveParcel @0x680cb0: COutPacket(&pkt,0x47) @0x680d75;
//	  Encode1(4) @0x680d88; Encode4(parcelId) @0x680d9d -- gated on the
//	  30-day expiry countdown @0x680d53 (parcel.go +21 field doc).
//	CTabReceive::DiscardParcel @0x680e20: COutPacket(&pkt,0x47) @0x680e9f;
//	  Encode1(5) @0x680eb2; Encode4(parcelId) @0x680ec7.
//	CParcelDlg::CloseParcelDlg @0x680740: COutPacket(&pkt,0x47) @0x68076c;
//	  Encode1(7) @0x68077f; no further bytes.
//
// COutPacket::COutPacket(&pkt, 0x47u) confirms opcode 71/0x047 at every call
// site, matching status.json's recorded gms_v92 opcode 71 and
// duey_action.yaml's own provenance. Byte-identical shape to v83/v84/v87.
//
// The mode byte itself is written by the shared Action wrapper (action.go),
// not by these per-op structs, so it is not part of the byte sequences
// pinned below (mirrors the existing action_test.go discipline). Full
// per-op read orders are recorded in the spliced export entries
// (docs/packets/ida-exports/gms_v92.json) and in the generated audit
// reports (docs/packets/audits/gms_v92/ParcelAction*.json) -- all four
// report verdicts are VerdictMatch (0) with FlatInvalid=false.

// packet-audit:verify packet=parcel/serverbound/ParcelActionSend version=gms_v92 ida=0x682da0
func TestActionSendV92(t *testing.T) {
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

// packet-audit:verify packet=parcel/serverbound/ParcelActionReceive version=gms_v92 ida=0x680cb0
func TestActionReceiveV92(t *testing.T) {
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

// packet-audit:verify packet=parcel/serverbound/ParcelActionDiscard version=gms_v92 ida=0x680e20
func TestActionDiscardV92(t *testing.T) {
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

// packet-audit:verify packet=parcel/serverbound/ParcelActionClose version=gms_v92 ida=0x680740
func TestActionCloseV92(t *testing.T) {
	l, ctx := newTestLogCtx()
	var a ActionClose
	got := a.Encode(l, ctx)(nil)
	if len(got) != 0 {
		t.Errorf("got % x want empty (mode written by the Action wrapper, not ActionClose)", got)
	}
}
