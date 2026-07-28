package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/serverbound/FieldItcQueryCashRequest version=gms_v83 ida=0x59eece
//
// ITC_QUERY_CASH_REQUEST (gms_v83 serverbound opcode 0xFC/252). Derived from
// CITC::TrySendQueryCashRequest @0x59eece (MapleStory_dump.exe v83 Me, IDA port
// 13342). The send site is:
//
//	if ( this[6] ) return 0;            // 0x59eede — m_bITCRequestSent latch
//	COutPacket::COutPacket(v3, 0xFC);   // 0x59eef0 — opcode 0xFC, no field writes
//	CClientSocket::SendPacket(..., v3); // 0x59ef03 — sent immediately
//	this[6] = 1;                        // 0x59ef0f
//	ZArray<unsigned char>::RemoveAll(v4); // 0x59ef16
//
// ZERO Encode calls between the COutPacket constructor (opcode only) and
// SendPacket → the packet body is empty (bodiless / opcode-only). The latch
// only prevents a double-send; it does not write to the wire.
func TestItcQueryCashRequestByteOutput(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	got := ItcQueryCashRequest{}.Encode(nil, ctx)(nil)
	if len(got) != 0 {
		t.Fatalf("ItcQueryCashRequest body (v83): got %d bytes %v, want 0 bytes (bodiless)", len(got), got)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcQueryCashRequest version=gms_v84 ida=0x5af26a
//
// ITC_QUERY_CASH_REQUEST (gms_v84 serverbound opcode 0x103/259). Derived from
// CITC::TrySendQueryCashRequest @0x5af26a (GMS_v84.1_U_DEVM.exe, IDA port
// 13337). The send site is structurally identical to the v83 twin @0x59eece
// except the opcode immediate, and sits at the exact same intra-class offset
// from CITC::OnStatusCharge (0x5af26a-0x5aef76 == 0x59eece-0x59ebda == 0x2F4):
//
//	cmp dword ptr [esi+18h], 0          // 0x5af27a — m_bITCRequestSent latch
//	push 103h                           // 0x5af284 — opcode 0x103, no field writes
//	call COutPacket::COutPacket         // 0x5af28c
//	call CClientSocket::SendPacket      // 0x5af29f — sent immediately
//	mov dword ptr [esi+18h], 1          // 0x5af2ab
//	call ZArray_RemoveAll               // 0x5af2b2
//
// The CSV/registry's 0xFC was the v83 opcode carried over unshifted (the CSVs
// have no v84 column); corrected to 0x103 in a prior commit
// (TestItcQueryCashRequestV84Opcode). ZERO Encode calls between the ctor and
// SendPacket → empty body.
func TestItcQueryCashRequestByteOutput_v84(t *testing.T) {
	ctx := pt.CreateContext("GMS", 84, 1)
	got := ItcQueryCashRequest{}.Encode(nil, ctx)(nil)
	if len(got) != 0 {
		t.Fatalf("ItcQueryCashRequest body (v84): got %d bytes %v, want 0 bytes (bodiless)", len(got), got)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcQueryCashRequest version=gms_v87 ida=0x5cec92
//
// ITC_QUERY_CASH_REQUEST (gms_v87 serverbound opcode 0x10A/266). Derived from
// CITC::TrySendQueryCashRequest @0x5cec92 (GMSv87_4GB.exe, IDA port 13341):
//
//	if ( this[6] ) return 0;             // 0x5ceca2 — m_bITCRequestSent latch
//	COutPacket::COutPacket(&a3, 0x10A);  // 0x5cecb4 — opcode 0x10A, no field writes
//	CClientSocket::SendPacket(..., &a3); // 0x5cecc7 — sent immediately
//	this[6] = 1;                         // 0x5cecd3
//	ZArray<unsigned char>::RemoveAll(v5);// 0x5cecda
//
// ZERO Encode calls between the ctor and SendPacket → empty body.
func TestItcQueryCashRequestByteOutput_v87(t *testing.T) {
	ctx := pt.CreateContext("GMS", 87, 1)
	got := ItcQueryCashRequest{}.Encode(nil, ctx)(nil)
	if len(got) != 0 {
		t.Fatalf("ItcQueryCashRequest body (v87): got %d bytes %v, want 0 bytes (bodiless)", len(got), got)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcQueryCashRequest version=gms_v95 ida=0x572ad0
//
// ITC_QUERY_CASH_REQUEST (gms_v95 serverbound opcode 0x133/307). Derived from
// CITC::TrySendQueryCashRequest @0x572ad0 (GMS_v95.0_U_DEVM.exe, IDA port
// 13340):
//
//	if ( this->m_bITCRequestSent ) return 0;  // 0x572af6 — named latch field
//	COutPacket::COutPacket(&oPacket, 307);    // 0x572b18 — opcode 0x133, no field writes
//	CClientSocket::SendPacket(..., &oPacket); // 0x572b30 — sent immediately
//	this->m_bITCRequestSent = 1;              // 0x572b39
//	ZArray<unsigned char>::RemoveAll(&v5);    // 0x572b48
//
// The latch field is named m_bITCRequestSent in this build (PDB), confirming
// the this[6] semantics in the other versions. ZERO Encode calls between the
// ctor and SendPacket → empty body.
func TestItcQueryCashRequestByteOutput_v95(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 0)
	got := ItcQueryCashRequest{}.Encode(nil, ctx)(nil)
	if len(got) != 0 {
		t.Fatalf("ItcQueryCashRequest body (v95): got %d bytes %v, want 0 bytes (bodiless)", len(got), got)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcQueryCashRequest version=jms_v185 ida=0x6043bf
//
// ITC_QUERY_CASH_REQUEST (jms_v185 serverbound opcode 0x10B/267). Derived from
// CITC::TrySendQueryCashRequest @0x6043bf (MapleStory_dump_SCY.exe jms_v185
// *_U_DEVM build, IDA port 13339):
//
//	if ( *(this + 6) ) return 0;          // 0x6043cf — m_bITCRequestSent latch
//	COutPacket::COutPacket(v4, 0x10B);    // 0x6043e1 — opcode 0x10B, no field writes
//	CClientSocket::SendPacket(..., v4);   // 0x6043f4 — sent immediately
//	*(this + 6) = 1;                      // 0x604400
//	ZArray<unsigned char>::RemoveAll(&v5);// 0x604407
//
// ZERO Encode calls between the ctor and SendPacket → empty body.
func TestItcQueryCashRequestByteOutput_jms185(t *testing.T) {
	ctx := pt.CreateContext("JMS", 185, 1)
	got := ItcQueryCashRequest{}.Encode(nil, ctx)(nil)
	if len(got) != 0 {
		t.Fatalf("ItcQueryCashRequest body (jms_v185): got %d bytes %v, want 0 bytes (bodiless)", len(got), got)
	}
}

func TestItcQueryCashRequestRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ItcQueryCashRequest{}
			output := ItcQueryCashRequest{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}
