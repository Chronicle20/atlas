package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestPetNameChangedBytesV48: CPet::OnNameChanged @0x58da70 (v48 IDB
// GMS_v48_1_DEVM.exe.i64) does DecodeStr(name) then
// `if (CInPacket::Decode1(a2))` selects the name-tag decoration layer, same
// str+byte shape as every other GMS version. NOT n-a: CUser::OnPetPacket
// @0x69221b case 'q' (113/0x071) dispatches here — the registry previously
// had no PET_NAMECHANGE entry for v48 at all (task-224 added one).
// packet-audit:verify packet=pet/clientbound/PetNameChanged version=gms_v48 ida=0x58da70
func TestPetNameChangedBytesV48(t *testing.T) {
	ctx := test.CreateContext("GMS", 48, 1)
	got := NewPetNameChanged(0x01020304, 0x05, "Fluffy").Encode(nil, ctx)(nil)
	want := []byte{
		0x04, 0x03, 0x02, 0x01,
		0x05,
		0x06, 0x00,
		'F', 'l', 'u', 'f', 'f', 'y',
		0x00,
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v48 = % X, want % X", got, want)
	}
}

// TestPetNameChangedBytesV61: CPet::OnNameChanged @0x6135d6 (v61 IDB
// GMS_v61.1_U_DEVM.exe.i64) does DecodeStr(name) then
// `if ((unsigned __int8)CInPacket::Decode1(a2))`, same str+byte shape.
// packet-audit:verify packet=pet/clientbound/PetNameChanged version=gms_v61 ida=0x6135d6
func TestPetNameChangedBytesV61(t *testing.T) {
	ctx := test.CreateContext("GMS", 61, 1)
	got := NewPetNameChanged(0x01020304, 0x05, "Fluffy").Encode(nil, ctx)(nil)
	want := []byte{
		0x04, 0x03, 0x02, 0x01,
		0x05,
		0x06, 0x00,
		'F', 'l', 'u', 'f', 'f', 'y',
		0x00,
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v61 = % X, want % X", got, want)
	}
}

// TestPetNameChangedBytesV72: CPet::OnNameChanged @0x66c137 (v72 IDB
// GMS_v72.1_U_DEVM.exe.i64) does DecodeStr(name) then
// `if (CInPacket::Decode1(a2))`, same str+byte shape.
// packet-audit:verify packet=pet/clientbound/PetNameChanged version=gms_v72 ida=0x66c137
func TestPetNameChangedBytesV72(t *testing.T) {
	ctx := test.CreateContext("GMS", 72, 1)
	got := NewPetNameChanged(0x01020304, 0x05, "Fluffy").Encode(nil, ctx)(nil)
	want := []byte{
		0x04, 0x03, 0x02, 0x01,
		0x05,
		0x06, 0x00,
		'F', 'l', 'u', 'f', 'f', 'y',
		0x00,
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v72 = % X, want % X", got, want)
	}
}

// TestPetNameChangedBytesV79: CPet::OnNameChanged @0x690f7f (v79 IDB
// GMS_v79_1_DEVM.exe.i64) does DecodeStr(name) then
// `if (CInPacket::Decode1(a2))`, same str+byte shape. The registry's prior
// address (8987764/0x892474) pointed at the shared CUser::OnPetPacket
// dispatcher, not this leaf handler; task-224 corrected it.
// packet-audit:verify packet=pet/clientbound/PetNameChanged version=gms_v79 ida=0x690f7f
func TestPetNameChangedBytesV79(t *testing.T) {
	ctx := test.CreateContext("GMS", 79, 1)
	got := NewPetNameChanged(0x01020304, 0x05, "Fluffy").Encode(nil, ctx)(nil)
	want := []byte{
		0x04, 0x03, 0x02, 0x01,
		0x05,
		0x06, 0x00,
		'F', 'l', 'u', 'f', 'f', 'y',
		0x00,
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v79 = % X, want % X", got, want)
	}
}

// TestPetNameChangedBytesV83: CPet::OnNameChanged @0x704801 does
// DecodeStr(name) -> this+38, then `if (CInPacket::Decode1(a2))` selects the
// name-tag decoration layer passed to CLife::MakeNameTag.
// packet-audit:verify packet=pet/clientbound/PetNameChanged version=gms_v83 ida=0x704801
func TestPetNameChangedBytesV83(t *testing.T) {
	ctx := test.CreateContext("GMS", 83, 1)
	got := NewPetNameChanged(0x01020304, 0x05, "Fluffy").Encode(nil, ctx)(nil)
	want := []byte{
		0x04, 0x03, 0x02, 0x01, // ownerId (upstream)
		0x05,       // slot (upstream)
		0x06, 0x00, // name length
		'F', 'l', 'u', 'f', 'f', 'y',
		0x00, // nameTag layer selector (GMS only)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v83 = % X, want % X", got, want)
	}
}

// TestPetNameChangedBytesV84: CPet::OnNameChanged @0x720f26 (v84 IDB
// GMS_v84.1_U_DEVM.i64) does DecodeStr(name) then `if (CInPacket::Decode1(a2))`,
// same str+byte shape (structurally identical to v72/v79/v83/v87). The
// function was unnamed (sub_720F26 / registry placeholder
// CPet__OnNameChanged_recv_0xB0); task-224 named it CPet::OnNameChanged in
// the IDB.
// packet-audit:verify packet=pet/clientbound/PetNameChanged version=gms_v84 ida=0x720f26
func TestPetNameChangedBytesV84(t *testing.T) {
	ctx := test.CreateContext("GMS", 84, 1)
	got := NewPetNameChanged(0x01020304, 0x05, "Fluffy").Encode(nil, ctx)(nil)
	want := []byte{
		0x04, 0x03, 0x02, 0x01,
		0x05,
		0x06, 0x00,
		'F', 'l', 'u', 'f', 'f', 'y',
		0x00,
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v84 = % X, want % X", got, want)
	}
}

// TestPetNameChangedBytesV87: CPet::OnNameChanged @0x7484e0 (v87 IDB
// GMSv87_4GB.exe.i64) does DecodeStr(name) then `if (CInPacket::Decode1(a2))`,
// then CLife::MakeNameTag directly (named function; PDB-style symbols
// already present in this IDB), same str+byte shape.
// packet-audit:verify packet=pet/clientbound/PetNameChanged version=gms_v87 ida=0x7484e0
func TestPetNameChangedBytesV87(t *testing.T) {
	ctx := test.CreateContext("GMS", 87, 1)
	got := NewPetNameChanged(0x01020304, 0x05, "Fluffy").Encode(nil, ctx)(nil)
	want := []byte{
		0x04, 0x03, 0x02, 0x01,
		0x05,
		0x06, 0x00,
		'F', 'l', 'u', 'f', 'f', 'y',
		0x00,
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v87 = % X, want % X", got, want)
	}
}

// TestPetNameChangedBytesV92: CPet::OnNameChanged @0x6967c0 (v92 IDB
// GMS_v92_1_DEVM.exe.i64) does DecodeStr(name) then
// `if (CInPacket::Decode1((int)v3))`, same str+byte shape. The whole v92 pet
// column was unverified entering this task (design note) — this is a fresh
// derivation, not a copy of a neighbour.
// packet-audit:verify packet=pet/clientbound/PetNameChanged version=gms_v92 ida=0x6967c0
func TestPetNameChangedBytesV92(t *testing.T) {
	ctx := test.CreateContext("GMS", 92, 1)
	got := NewPetNameChanged(0x01020304, 0x05, "Fluffy").Encode(nil, ctx)(nil)
	want := []byte{
		0x04, 0x03, 0x02, 0x01,
		0x05,
		0x06, 0x00,
		'F', 'l', 'u', 'f', 'f', 'y',
		0x00,
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v92 = % X, want % X", got, want)
	}
}

// TestPetNameChangedBytesV95: CPet::OnNameChanged @0x6a11f0 (v95 IDB
// GMS_v95.0_U_DEVM.exe.i64, PDB-backed) does DecodeStr(name) then
// `if (CInPacket::Decode1(v3)) nNameTag = this->m_pTemplate->nNameTag; else
// nNameTag = 0;` -- the named field access this codec's NameTagLayer doc
// comment cites, same str+byte shape.
// packet-audit:verify packet=pet/clientbound/PetNameChanged version=gms_v95 ida=0x6a11f0
func TestPetNameChangedBytesV95(t *testing.T) {
	ctx := test.CreateContext("GMS", 95, 1)
	got := NewPetNameChanged(0x01020304, 0x05, "Fluffy").Encode(nil, ctx)(nil)
	want := []byte{
		0x04, 0x03, 0x02, 0x01,
		0x05,
		0x06, 0x00,
		'F', 'l', 'u', 'f', 'f', 'y',
		0x00,
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v95 = % X, want % X", got, want)
	}
}

// TestPetNameChangedBytesJMS185 is the regression test for design C-1: JMS
// v185's CPet::OnNameChanged @0x76a5de performs exactly one DecodeStr and NO
// Decode1 — it branches on sub_768D82(this), a client-side state query, not a
// wire byte. The JMS body is therefore one byte SHORTER than the GMS body.
// packet-audit:verify packet=pet/clientbound/PetNameChanged version=jms_v185 ida=0x76a5de
func TestPetNameChangedBytesJMS185(t *testing.T) {
	ctx := test.CreateContext("JMS", 185, 1)
	got := NewPetNameChanged(0x01020304, 0x05, "Fluffy").Encode(nil, ctx)(nil)
	want := []byte{
		0x04, 0x03, 0x02, 0x01,
		0x05,
		0x06, 0x00,
		'F', 'l', 'u', 'f', 'f', 'y',
		// no trailing flag byte
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("jms_v185 = % X, want % X", got, want)
	}
	if len(got) != 13 {
		t.Fatalf("jms body length = %d, want 13 (one shorter than GMS)", len(got))
	}
}

// TestPetNameChangedDecodeRoundTripGMS uses the shared test.RoundTrip helper
// (libs/atlas-packet/test/roundtrip.go) rather than an inline reader — it is
// what every other codec in this package uses for decode coverage, and it
// additionally asserts no unconsumed bytes remain after decode.
func TestPetNameChangedDecodeRoundTripGMS(t *testing.T) {
	ctx := test.CreateContext("GMS", 83, 1)
	input := NewPetNameChanged(0x01020304, 0x05, "Rex")
	test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
}

// The rename packet and the spawn body must write the SAME decoration selector,
// or a renamed pet's name tag appears on rename and vanishes on the next
// respawn (design §1, "what the clientbound flag byte selects").
func TestNameTagLayerAgreesWithActivated(t *testing.T) {
	ctx := test.CreateContext("GMS", 83, 1)
	spawn := NewPetSpawnActivated(1, 0, 5000000, "Rex", 7, 0, 0, 0, 0).Encode(nil, ctx)(nil)
	// nameTag is the second-to-last byte of the active spawn body, immediately
	// before chatBalloon (activated.go Encode).
	if spawn[len(spawn)-2] != NameTagLayer {
		t.Fatalf("Activated nameTag = %d, want NameTagLayer (%d)", spawn[len(spawn)-2], NameTagLayer)
	}
}
