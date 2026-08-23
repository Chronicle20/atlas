package serverbound

import (
	"bytes"
	"testing"
)

// v95 DUEY_ACTION (op 0x046) verification (task-241 Task 28 batch 7/8,
// session ecc757f4, GMS_v95.0_U_DEVM.exe.i64). Every byte cites this IDB's
// own decompile directly:
//
//	CTabSend::SendParcel @0x690140: NPC path. COutPacket(&pkt,70) @0x6902b8;
//	  Encode1(2) mode @0x6902c8; Encode1(invType=m_nTI) @0x6902d6;
//	  Encode2(slot=m_nPOS) @0x6902e4; Encode2(quantity=m_nNumber) @0x6902f2;
//	  Encode4(mesos) @0x6902ff; EncodeStr(recipientName) @0x69031b;
//	  Encode1(0) @0x690325 quick=false -- stops there.
//	CTabQuickSend::SendQuickDelivery @0x68fa00: quick path (named directly in
//	  THIS IDB, unlike v92's unnamed sub_682DA0). COutPacket(&pkt,70)
//	  @0x68fa06; Encode1(2) mode @0x68fa16; Encode1(invType) @0x68fa24;
//	  Encode2(slot) @0x68fa32; Encode2(quantity) @0x68fa40; Encode4(mesos)
//	  @0x68fa4d; EncodeStr(recipientName) @0x68fa69; Encode1(1) @0x68fa74
//	  quick=true; EncodeStr(message=sMemo) @0x68fa90; Encode4(ticketRef=
//	  m_nCashPOS) @0x68fa9d.
//	CTabReceive::ReceiveParcel @0x68f470: COutPacket(&pkt,70) @0x68f535;
//	  Encode1(4) @0x68f548; Encode4(parcelId=nSN) @0x68f55d -- gated on the
//	  30-day expiry countdown @0x68f513 (parcel.go +21 field doc).
//	CTabReceive::DiscardParcel @0x68f5e0: COutPacket(&pkt,70) @0x68f65f;
//	  Encode1(5) @0x68f672; Encode4(parcelId=nSN) @0x68f687.
//	CParcelDlg::CloseParcelDlg @0x68ef40: COutPacket(&pkt,70) @0x68ef6c;
//	  Encode1(7) @0x68ef7f; no further bytes.
//
// COutPacket::COutPacket(&pkt, 70) confirms opcode 70/0x046 at every call
// site, matching status.json's recorded gms_v95 opcode 70 and
// duey_action.yaml's own provenance. Byte-identical shape to v83/v84/v87/v92.
//
// Note: a sixth call site sharing op 0x046, CUIFadeYesNo::OnButtonClicked
// case 10 @0x529f04 (Encode1(0) mode=0; Encode4(0xFFFFFFFF); Encode4(2)), is
// the "accept parcel alarm" response to ALARM_NAMED/ALARM_GENERIC -- it is a
// DIFFERENT DUEY_ACTION mode (0, not one of SEND/RECEIVE/DISCARD/CLOSE) and
// is out of scope for this batch's four target arms; confirmed present in
// this roster's fname list (docs/packets/dispatchers/duey_action.yaml) but
// not verified here.
//
// The mode byte itself is written by the shared Action wrapper (action.go),
// not by these per-op structs, so it is not part of the byte sequences
// pinned below (mirrors the existing action_test.go discipline). Full
// per-op read orders are recorded in the spliced export entries
// (docs/packets/ida-exports/gms_v95.json) and in the generated audit
// reports (docs/packets/audits/gms_v95/ParcelAction*.json) -- all four
// report verdicts are VerdictMatch (0) with FlatInvalid=false.

// packet-audit:verify packet=parcel/serverbound/ParcelActionSend version=gms_v95 ida=0x690140
func TestActionSendV95(t *testing.T) {
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

// packet-audit:verify packet=parcel/serverbound/ParcelActionReceive version=gms_v95 ida=0x68f470
func TestActionReceiveV95(t *testing.T) {
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

// packet-audit:verify packet=parcel/serverbound/ParcelActionDiscard version=gms_v95 ida=0x68f5e0
func TestActionDiscardV95(t *testing.T) {
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

// packet-audit:verify packet=parcel/serverbound/ParcelActionClose version=gms_v95 ida=0x68ef40
func TestActionCloseV95(t *testing.T) {
	l, ctx := newTestLogCtx()
	var a ActionClose
	got := a.Encode(l, ctx)(nil)
	if len(got) != 0 {
		t.Errorf("got % x want empty (mode written by the Action wrapper, not ActionClose)", got)
	}
}
