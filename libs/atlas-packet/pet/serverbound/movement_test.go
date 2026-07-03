package serverbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=pet/serverbound/PetMovementRequest version=gms_v83 ida=0x9c4e41
// packet-audit:verify packet=pet/serverbound/PetMovementRequest version=gms_v87 ida=0xa558b6
// packet-audit:verify packet=pet/serverbound/PetMovementRequest version=gms_v95 ida=0x99f5a0
// packet-audit:verify packet=pet/serverbound/PetMovementRequest version=jms_v185 ida=0xaa25ab
// packet-audit:verify packet=pet/serverbound/PetMovementRequest version=gms_v84 ida=0xa0c600
func TestPetMovement(t *testing.T) {
	p := MovementRequest{}
	p.petId = 5000001

	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, p.Encode, p.Decode, nil)

			if p.PetId() != 5000001 {
				t.Errorf("expected petId 5000001, got %d", p.PetId())
			}
			if p.PetIdAsUint32() != 5000001 {
				t.Errorf("expected petIdAsUint32 5000001, got %d", p.PetIdAsUint32())
			}
		})
	}
}

// v79 MOVE_PET (sb op 163=0xA3) send order, verified GMS_v79_1_DEVM.exe (port
// 13340): sub_9150A1 — COutPacket(163)@0x9150cd, EncodeBuffer(petId,8)@0x9150ef,
// then CMovePath::Flush (opaque movement). Wire = petId(8)+movement; empty
// model.Movement = StartX(2)+StartY(2)+count(1) = 5 zero bytes. Identical to v83.
// TestPetMovementBytesV72 pins the v72 wire = v79 (no version gate). IDA
// GMS_v72.1_U_DEVM.exe @port 13339: CVecCtrlPet::EndUpdateActive@0x8c516e builds
// COutPacket(161)@0x8c519a, EncodeBuffer(petId,8)@0x8c51bc, then CMovePath::Flush
// writes the raw movement blob.
// packet-audit:verify packet=pet/serverbound/PetMovementRequest version=gms_v72 ida=0x8c516e
func TestPetMovementBytesV72(t *testing.T) {
	ctx := test.CreateContext("GMS", 72, 1)
	p := MovementRequest{petId: 0x0102030405060708}
	got := p.Encode(nil, ctx)(nil)
	want := []byte{
		0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01, // petId EncodeBuffer(8)@0x8c51bc (LE)
		0x00, 0x00, // movement StartX
		0x00, 0x00, // movement StartY
		0x00,       // movement element count = 0
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v72 = % X, want % X", got, want)
	}
}

// TestPetMovementBytesV61 pins the v61 wire = v72 (no version gate). IDA
// GMS_v61.1_U_DEVM.exe @port 13338: CVecCtrlPet::EndUpdateActive sub_7FBB76@0x7fbb76
// builds COutPacket(138)@0x7fbba5, EncodeBuffer(petId,8)@0x7fbbc6, then
// CMovePath::Flush writes the raw movement blob. v72 op161 (Δ-23).
// packet-audit:verify packet=pet/serverbound/PetMovementRequest version=gms_v61 ida=0x7fbb76
func TestPetMovementBytesV61(t *testing.T) {
	ctx := test.CreateContext("GMS", 61, 1)
	p := MovementRequest{petId: 0x0102030405060708}
	got := p.Encode(nil, ctx)(nil)
	want := []byte{
		0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01, // petId EncodeBuffer(8)@0x7fbbc6 (LE)
		0x00, 0x00, // movement StartX
		0x00, 0x00, // movement StartY
		0x00, // movement element count = 0
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v61 = % X, want % X", got, want)
	}
}

// packet-audit:verify packet=pet/serverbound/PetMovementRequest version=gms_v79 ida=0x9150a1
func TestPetMovementBytesV79(t *testing.T) {
	ctx := test.CreateContext("GMS", 79, 1)
	p := MovementRequest{petId: 0x0102030405060708}
	got := p.Encode(nil, ctx)(nil)
	want := []byte{
		0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01, // petId EncodeBuffer(8)@0x9150ef (LE)
		0x00, 0x00, // movement StartX
		0x00, 0x00, // movement StartY
		0x00,       // movement element count = 0
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v79 = % X, want % X", got, want)
	}
}

func TestPetMovementOperationString(t *testing.T) {
	p := MovementRequest{}
	if p.Operation() != PetMovementHandle {
		t.Errorf("expected operation %s, got %s", PetMovementHandle, p.Operation())
	}
	if p.String() == "" {
		t.Error("expected non-empty string")
	}
}

// TestPetMovementBytesV48 pins the v48 MOVE_PET (sb op 113 / 0x71) send. IDA
// GMS_v48_1_DEVM.exe @port 13337: sub_6E5BD6@0x6e5bff builds COutPacket(113)
// then sub_5622DA (CMovePath::Flush) writes the raw movement blob — NO leading
// EncodeBuffer(petId,8) (v48 single-pet; hasLeadingPetId(GMS,48)=false). Empty
// model.Movement = StartX(2)+StartY(2)+count(1) = 5 zero bytes. v61 op138 carries petId.
// packet-audit:verify packet=pet/serverbound/PetMovementRequest version=gms_v48 ida=0x6e5bd6
func TestPetMovementBytesV48(t *testing.T) {
	ctx := test.CreateContext("GMS", 48, 1)
	p := MovementRequest{petId: 0x0102030405060708}
	got := p.Encode(nil, ctx)(nil)
	want := []byte{
		// NO petId on v48 (single-pet)
		0x00, 0x00, // movement StartX
		0x00, 0x00, // movement StartY
		0x00, // movement element count = 0
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v48 = % X, want % X", got, want)
	}
}
