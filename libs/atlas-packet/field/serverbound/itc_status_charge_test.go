package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/serverbound/FieldItcStatusCharge version=gms_v83 ida=0x59ebda
//
// ITC_STATUS_CHARGE (gms_v83 serverbound opcode 0xFB/251). Derived from
// CITC::OnStatusCharge @0x59ebda (MapleStory_dump.exe v83 Me, IDA port 13342).
// The send site is:
//
//	cmp dword ptr [ecx+18h], 0  // 0x59ebe7 — m_bITCRequestSent latch
//	mov dword ptr [ecx+18h], 1  // 0x59ebed
//	push 0FBh                   // 0x59ebf4 — opcode 0xFB, no field writes
//	call COutPacket::COutPacket // 0x59ebfc
//	call CClientSocket::SendPacket // 0x59ec0f — sent immediately
//	call ZArray<unsigned char>::RemoveAll // 0x59ec1b
//
// ZERO Encode calls between the COutPacket constructor (opcode only) and
// SendPacket → the packet body is empty (bodiless / opcode-only). The latch
// only prevents a double-send; it does not write to the wire.
func TestItcStatusChargeByteOutput(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	got := ItcStatusCharge{}.Encode(nil, ctx)(nil)
	if len(got) != 0 {
		t.Fatalf("ItcStatusCharge body (v83): got %d bytes %v, want 0 bytes (bodiless)", len(got), got)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcStatusCharge version=gms_v84 ida=0x5aef76
//
// ITC_STATUS_CHARGE (gms_v84 serverbound opcode 0x102/258). Derived from
// CITC::OnStatusCharge @0x5aef76 (GMS_v84.1_U_DEVM.exe, IDA port 13337). The
// send site is byte-for-byte identical to the v83 twin @0x59ebda except the
// opcode immediate:
//
//	cmp dword ptr [ecx+18h], 0  // 0x5aef83 — m_bITCRequestSent latch
//	mov dword ptr [ecx+18h], 1  // 0x5aef89
//	push 102h                   // 0x5aef90 — opcode 0x102, no field writes
//	call COutPacket::COutPacket // 0x5aef98
//	call CClientSocket::SendPacket // 0x5aefab — sent immediately
//	call ZArray_RemoveAll       // 0x5aefb7
//
// The CSV/registry's 0xFB was the v83 opcode carried over unshifted; corrected
// to 0x102 in a prior commit (TestItcStatusChargeV84Opcode). ZERO Encode calls
// between the ctor and SendPacket → empty body.
func TestItcStatusChargeByteOutput_v84(t *testing.T) {
	ctx := pt.CreateContext("GMS", 84, 1)
	got := ItcStatusCharge{}.Encode(nil, ctx)(nil)
	if len(got) != 0 {
		t.Fatalf("ItcStatusCharge body (v84): got %d bytes %v, want 0 bytes (bodiless)", len(got), got)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcStatusCharge version=gms_v87 ida=0x5ce90b
//
// ITC_STATUS_CHARGE (gms_v87 serverbound opcode 0x109/265). Derived from
// CITC::OnStatusCharge @0x5ce90b (GMSv87_4GB.exe, IDA port 13341):
//
//	if ( !this[6] )                       // 0x5ce918 — m_bITCRequestSent latch
//	    this[6] = 1;                      // 0x5ce91e
//	COutPacket::COutPacket(&a3, 0x109);   // 0x5ce92d — opcode 0x109, no field writes
//	CClientSocket::SendPacket(..., &a3);  // 0x5ce940 — sent immediately
//	ZArray<unsigned char>::RemoveAll(v3); // 0x5ce94c
//
// ZERO Encode calls between the ctor and SendPacket → empty body.
func TestItcStatusChargeByteOutput_v87(t *testing.T) {
	ctx := pt.CreateContext("GMS", 87, 1)
	got := ItcStatusCharge{}.Encode(nil, ctx)(nil)
	if len(got) != 0 {
		t.Fatalf("ItcStatusCharge body (v87): got %d bytes %v, want 0 bytes (bodiless)", len(got), got)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcStatusCharge version=gms_v95 ida=0x572a50
//
// ITC_STATUS_CHARGE (gms_v95 serverbound opcode 0x132/306). Derived from
// CITC::OnStatusCharge @0x572a50 (GMS_v95.0_U_DEVM.exe, IDA port 13340):
//
//	if ( !this->m_bITCRequestSent )           // 0x572a73 — named latch field
//	    this->m_bITCRequestSent = 1;          // 0x572a79
//	COutPacket::COutPacket(&oPacket, 306);    // 0x572a89 — opcode 0x132, no field writes
//	CClientSocket::SendPacket(..., &oPacket); // 0x572aa1 — sent immediately
//	ZArray<unsigned char>::RemoveAll(&v3);    // 0x572ab2
//
// The latch field is named m_bITCRequestSent in this build (PDB), confirming
// the this[6] semantics in the other versions. ZERO Encode calls between the
// ctor and SendPacket → empty body.
func TestItcStatusChargeByteOutput_v95(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 0)
	got := ItcStatusCharge{}.Encode(nil, ctx)(nil)
	if len(got) != 0 {
		t.Fatalf("ItcStatusCharge body (v95): got %d bytes %v, want 0 bytes (bodiless)", len(got), got)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcStatusCharge version=jms_v185 ida=0x6040a9
//
// ITC_STATUS_CHARGE (jms_v185 serverbound opcode 0x10A/266). Derived from
// CITC::OnStatusCharge @0x6040a9 (MapleStory_dump_SCY.exe jms_v185 *_U_DEVM
// build, IDA port 13339):
//
//	if ( !*(this + 6) )                   // 0x6040b6 — m_bITCRequestSent latch
//	    *(this + 6) = 1;                  // 0x6040bc
//	COutPacket::COutPacket(v2, 0x10A);    // 0x6040cb — opcode 0x10A, no field writes
//	CClientSocket::SendPacket(..., v2);   // 0x6040de — sent immediately
//	ZArray<unsigned char>::RemoveAll(&v3);// 0x6040ea
//
// ZERO Encode calls between the ctor and SendPacket → empty body.
func TestItcStatusChargeByteOutput_jms185(t *testing.T) {
	ctx := pt.CreateContext("JMS", 185, 1)
	got := ItcStatusCharge{}.Encode(nil, ctx)(nil)
	if len(got) != 0 {
		t.Fatalf("ItcStatusCharge body (jms_v185): got %d bytes %v, want 0 bytes (bodiless)", len(got), got)
	}
}

func TestItcStatusChargeRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ItcStatusCharge{}
			output := ItcStatusCharge{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}
