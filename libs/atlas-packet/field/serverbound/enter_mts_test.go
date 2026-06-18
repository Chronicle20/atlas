package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/serverbound/FieldEnterMts version=gms_v83 ida=0xa12522
//
// ENTER_MTS (gms_v83 serverbound opcode 0x9C/156). Derived from
// CWvsContext::SendMigrateToITCRequest @0xa12522 (MapleStory_dump.exe v83 Me,
// IDA port 13342). The send site at 0xa1263b is:
//
//	COutPacket::COutPacket(v14, 0x9C);   // 0xa1263b — opcode 0x9C, no field writes
//	v18 = 2;
//	CClientSocket::SendPacket(..., v14); // 0xa12651 — sent immediately
//
// There are ZERO Encode calls between the COutPacket constructor (opcode only)
// and SendPacket. The packet body is therefore empty (bodiless / opcode-only).
// The preceding branches (guest-ID guard @0xa12532, "right now" guard
// @0xa12580, lie-detector guard @0xa125b8, map-flag guard @0xa1260b) all emit
// local CHATLOG/Notice text and return early — none of them write to the wire.
// Expected wire body (excluding the opcode the socket layer prepends): empty.
func TestEnterMtsByteOutput(t *testing.T) {
	// gms_v83 context (the verified version for this op).
	ctx := pt.CreateContext("GMS", 83, 1)
	input := EnterMts{}
	got := input.Encode(nil, ctx)(nil)
	// Bodiless: no bytes after the opcode (decompile @0xa1263b — COutPacket(0x9C)
	// then SendPacket with no intervening Encode).
	want := []byte{}
	if len(got) != len(want) {
		t.Fatalf("EnterMts body: got %d bytes %v, want %d bytes %v", len(got), got, len(want), want)
	}
}

func TestEnterMtsRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := EnterMts{}
			output := EnterMts{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}
