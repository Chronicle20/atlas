package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestAgreementResponseBytesV48 pins the gms_v48 guild-agreement body.
//
// IDA (GMS_v48_1_DEVM.exe): CField::SendCreateGuildAgreeMsg @0x4c5a18 builds
// COutPacket(96) @0x4c5a33 — opcode 96 — then Encode1(0x1E) @0x4c5a41 (the
// GUILD_OPERATION mode byte), Encode4(g_pWvsContext[2078]) @0x4c5a4f and
// Encode1(a2) @0x4c5a5a, and sends. The dispatcher strips the opcode and the
// mode byte, leaving Encode4(unk) + Encode1(agreed) = 5 bytes.
//
// This is the same shape the codec's doc comment already records for v83
// @0x530666, v87 @0x557e6e, v95 @0x52d780 and jms @0x56da47 — v48 uses mode
// 0x1E like v83/v87/jms (v95 shifted to 0x20). No wire change was needed.
//
// packet-audit:verify packet=guild/serverbound/GuildAgreementResponse version=gms_v48 ida=0x4c5a18
func TestAgreementResponseBytesV48(t *testing.T) {
	ctx := pt.CreateContext("GMS", 48, 1)
	in := AgreementResponse{unk: 0x01020304, agreed: true}

	got := in.Encode(nil, ctx)(nil)
	want := []byte{
		0x04, 0x03, 0x02, 0x01, // unk    — Encode4 @0x4c5a4f
		0x01, // agreed — Encode1 @0x4c5a5a
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v48 guild agreement body:\n got % x\nwant % x", got, want)
	}

	var out AgreementResponse
	pt.RoundTrip(t, ctx, in.Encode, out.Decode, nil)
}
