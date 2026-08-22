package serverbound

import (
	"bytes"
	"testing"
)

// v84 DUEY_ACTION (op 0x041) verification (task-241 Task 28, batch 4/8,
// session 46c2a2eb, GMS_v84.1_U_DEVM.i64). Every byte cites this IDB's own
// decompile directly:
//
//	CTabSend::SendParcel: two independent client call sites share the op,
//	  same asymmetric shape as v83.
//	  NPC path sub_70BB61 @0x70bb61: COutPacket(&pkt,0x41); Encode1(2) mode
//	  @0x70bc86; Encode1(invType)@0x70bc94; Encode2(slot)@0x70bca3;
//	  Encode2(quantity)@0x70bcb2; Encode4(mesos)@0x70bcbd;
//	  EncodeStr(recipientName)@0x70bcd6; Encode1(0)@0x70bcdf quick=false —
//	  stops there.
//	  Quick path sub_70A2AE @0x70a2ae: COutPacket(&pkt,0x41);
//	  Encode1(2)@0x70a464 mode; Encode1(invType)@0x70a472;
//	  Encode2(slot)@0x70a481; Encode2(quantity)@0x70a490;
//	  Encode4(mesos)@0x70a49b; EncodeStr(recipientName)@0x70a4b4;
//	  Encode1(1)@0x70a4be quick=true; EncodeStr(message)@0x70a4d7;
//	  Encode4(ticketRef)@0x70a4e2.
//	CTabReceive::ReceiveParcel @0x70915c: COutPacket(&pkt,0x41);
//	  Encode1(4)@0x70920c; Encode4(parcelId)@0x70921f — gated on the 30-day
//	  expiry countdown (same shape as v83).
//	CTabReceive::DiscardParcel @0x70927c: COutPacket(&pkt,65);
//	  Encode1(5)@0x7092f3; Encode4(parcelId)@0x709306.
//	CParcelDlg::CloseParcelDlg (sub_70DB46) @0x70db46: COutPacket(&pkt,65);
//	  Encode1(7)@0x70db69; no further bytes.
//
// The mode byte itself is written by the shared Action wrapper (action.go),
// not by these per-op structs, so it is not part of the byte sequences
// pinned below (mirrors the existing action_test.go/v83_test.go
// discipline). Full per-op read orders are recorded in the spliced export
// entries (docs/packets/ida-exports/gms_v84.json) and in the generated
// audit reports (docs/packets/audits/gms_v84/ParcelAction*.json).

// packet-audit:verify packet=parcel/serverbound/ParcelActionSend version=gms_v84 ida=0x70a2ae
func TestActionSendV84(t *testing.T) {
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

// packet-audit:verify packet=parcel/serverbound/ParcelActionReceive version=gms_v84 ida=0x70915c
func TestActionReceiveV84(t *testing.T) {
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

// packet-audit:verify packet=parcel/serverbound/ParcelActionDiscard version=gms_v84 ida=0x70927c
func TestActionDiscardV84(t *testing.T) {
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

// packet-audit:verify packet=parcel/serverbound/ParcelActionClose version=gms_v84 ida=0x70db46
func TestActionCloseV84(t *testing.T) {
	l, ctx := newTestLogCtx()
	var a ActionClose
	got := a.Encode(l, ctx)(nil)
	if len(got) != 0 {
		t.Errorf("got % x want empty (mode written by the Action wrapper, not ActionClose)", got)
	}
}
