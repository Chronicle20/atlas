package serverbound

import (
	"bytes"
	"testing"
)

// v79 DUEY_ACTION (op 0x03F) verification — task-241 Task 28, session
// f36df4cd (GMS_v79_1_DEVM.exe.i64). Every byte cites this IDB's own
// decompile directly:
//
//	CTabSend::SendParcel @0x68170f: NPC-path call site — COutPacket(&pkt,63)
//	  @0x681826; Encode1(2) mode @0x681834; Encode1(invType) @0x681842;
//	  Encode2(slot) @0x681851; Encode2(quantity) @0x681860; Encode4(mesos)
//	  @0x68186b; EncodeStr(recipientName) @0x681884; Encode1(0) @0x68188d
//	  quick=false — stops there. Quick path is a SEPARATE named function,
//	  CTabQuickSend::SendQuickDelivery @0x67fe5d: same mode/invType/slot/
//	  quantity/mesos/recipientName shape at offsets +8/+12/+20/+24 (this->
//	  layout differs slightly from the NPC struct), Encode1(1) @0x68006d
//	  quick=true, then EncodeStr(message) @0x680086 + Encode4(ticketRef)
//	  @0x680091.
//	CTabReceive::ReceiveParcel @0x67ed0c: COutPacket(&pkt,63);
//	  Encode1(4)@0x67edbc; Encode4(parcelId)@0x67edcf — gated on the 30-day
//	  expiry countdown (parcel.go +21 field doc).
//	CTabReceive::DiscardParcel @0x67ee2c: COutPacket(&pkt,63);
//	  Encode1(5)@0x67eea3; Encode4(parcelId)@0x67eeb6.
//	CParcelDlg::CloseParcelDlg @0x6836b0: COutPacket(&pkt,63);
//	  Encode1(7)@0x6836d3; no further bytes.
//
// The mode byte itself is written by the shared Action wrapper (action.go),
// not by these per-op structs, so it is not part of the byte sequences
// pinned below (mirrors the existing action_test.go discipline). Full
// per-op read orders are recorded in the spliced export entries
// (docs/packets/ida-exports/gms_v79.json) and in the generated audit
// reports (docs/packets/audits/gms_v79/ParcelAction*.json) — all four
// report verdicts are VerdictMatch (0) with FlatInvalid=false.

// packet-audit:verify packet=parcel/serverbound/ParcelActionSend version=gms_v79 ida=0x68170f
func TestActionSendV79(t *testing.T) {
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

// packet-audit:verify packet=parcel/serverbound/ParcelActionReceive version=gms_v79 ida=0x67ed0c
func TestActionReceiveV79(t *testing.T) {
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

// packet-audit:verify packet=parcel/serverbound/ParcelActionDiscard version=gms_v79 ida=0x67ee2c
func TestActionDiscardV79(t *testing.T) {
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

// packet-audit:verify packet=parcel/serverbound/ParcelActionClose version=gms_v79 ida=0x6836b0
func TestActionCloseV79(t *testing.T) {
	l, ctx := newTestLogCtx()
	var a ActionClose
	got := a.Encode(l, ctx)(nil)
	if len(got) != 0 {
		t.Errorf("got % x want empty (mode written by the Action wrapper, not ActionClose)", got)
	}
}
