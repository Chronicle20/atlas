package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

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
