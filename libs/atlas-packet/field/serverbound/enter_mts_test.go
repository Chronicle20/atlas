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

// packet-audit:verify packet=field/serverbound/FieldEnterMts version=gms_v84 ida=0xa5c95f
//
// ENTER_MTS (gms_v84 serverbound opcode 0xA0/160). Derived from
// CWvsContext::SendMigrateToITCRequest @0xa5c95f (GMS_v84.1_U_DEVM.exe, IDA
// port 13337). The send site at 0xa5ca78 is:
//
//	COutPacket::COutPacket((COutPacket *)v12, 160); // 0xa5ca78 — opcode 0xA0, no field writes
//	v16 = 2;
//	CClientSocket::SendPacket(..., v12);            // 0xa5ca8e — sent immediately
//
// ZERO Encode calls between the COutPacket constructor and SendPacket → empty
// body. Preceding branches (guest-ID guard @0xa5c96f, "right now" guard
// @0xa5c9bd, lie-detector guard @0xa5c9f5, map-flag guard @0xa5ca48) emit
// StringPool/Notice text and return early — none writes to the wire.
func TestEnterMtsByteOutput_v84(t *testing.T) {
	ctx := pt.CreateContext("GMS", 84, 1)
	got := EnterMts{}.Encode(nil, ctx)(nil)
	if len(got) != 0 {
		t.Fatalf("EnterMts body (v84): got %d bytes %v, want 0 bytes (bodiless)", len(got), got)
	}
}

// packet-audit:verify packet=field/serverbound/FieldEnterMts version=gms_v87 ida=0xaa7f49
//
// ENTER_MTS (gms_v87 serverbound opcode 0xA4/164). Derived from
// CWvsContext::SendMigrateToITCRequest @0xaa7f49 (GMSv87_4GB.exe, IDA port
// 13341). The send site at 0xaa8062 is:
//
//	COutPacket::COutPacket(&a3, 0xA4); // 0xaa8062 — opcode 0xA4, no field writes
//	v17 = 2;
//	CClientSocket::SendPacket(..., &a3); // 0xaa8078 — sent immediately
//
// ZERO Encode calls between the COutPacket constructor and SendPacket → empty
// body. Preceding branches (guest-ID guard @0xaa7f59, "right now" guard
// @0xaa7fa7, lie-detector guard @0xaa7fdf, map-flag guard @0xaa8032) emit
// CHATLOG_ADD/Notice text and return early — none writes to the wire.
func TestEnterMtsByteOutput_v87(t *testing.T) {
	ctx := pt.CreateContext("GMS", 87, 1)
	got := EnterMts{}.Encode(nil, ctx)(nil)
	if len(got) != 0 {
		t.Fatalf("EnterMts body (v87): got %d bytes %v, want 0 bytes (bodiless)", len(got), got)
	}
}

// packet-audit:verify packet=field/serverbound/FieldEnterMts version=gms_v95 ida=0x9def50
//
// ENTER_MTS (gms_v95 serverbound opcode 0xB4/180). Derived from
// CWvsContext::SendMigrateToITCRequest @0x9def50 (GMS_v95.0_U_DEVM.exe, IDA
// port 13340). The send site at 0x9df125 is:
//
//	COutPacket::COutPacket(&oPacket, 180); // 0x9df125 — opcode 0xB4, no field writes
//	v17 = 2;
//	CClientSocket::SendPacket(..., &oPacket); // 0x9df13d — sent immediately
//
// ZERO Encode calls between the COutPacket constructor and SendPacket → empty
// body. Preceding branches (m_bIsGuestAccount @0x9def76, "right now" guard
// @0x9df012, lie-detector guard @0x9df071, map-flag guard @0x9df0e0) emit
// ChatLogAdd/Notice text and return early — none writes to the wire.
func TestEnterMtsByteOutput_v95(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 0)
	got := EnterMts{}.Encode(nil, ctx)(nil)
	if len(got) != 0 {
		t.Fatalf("EnterMts body (v95): got %d bytes %v, want 0 bytes (bodiless)", len(got), got)
	}
}

// packet-audit:verify packet=field/serverbound/FieldEnterMts version=jms_v185 ida=0xaf860d
//
// ENTER_MTS (jms_v185 serverbound opcode 0xA6/166). Derived from
// CWvsContext::SendMigrateToITCRequest @0xaf860d (MapleStory_dump_SCY.exe
// jms_v185 *_U_DEVM build, IDA port 13339). The send site at 0xaf8752 is:
//
//	COutPacket::COutPacket(v19, 0xA6); // 0xaf8752 — opcode 0xA6, no field writes
//	v23 = 2;
//	CClientSocket::SendPacket(..., v19); // 0xaf8768 — sent immediately
//
// ZERO Encode calls between the COutPacket constructor and SendPacket → empty
// body. Preceding branches (CUserLocal-null @0xaf8627, "right now" guard
// @0xaf8633, terms-of-service YesNo/open_web_site2 @0xaf8653, lie-detector
// guard @0xaf86cf, map-flag guard @0xaf8722) emit dialog/Notice/web text and
// return early — none writes to the wire.
func TestEnterMtsByteOutput_jms185(t *testing.T) {
	ctx := pt.CreateContext("JMS", 185, 1)
	got := EnterMts{}.Encode(nil, ctx)(nil)
	if len(got) != 0 {
		t.Fatalf("EnterMts body (jms_v185): got %d bytes %v, want 0 bytes (bodiless)", len(got), got)
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
