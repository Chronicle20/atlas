package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=pet/clientbound/PetMovement version=gms_v83 ida=0x70474d
// packet-audit:verify packet=pet/clientbound/PetMovement version=gms_v87 ida=0x74842a
// packet-audit:verify packet=pet/clientbound/PetMovement version=gms_v95 ida=0x69fb60
// packet-audit:verify packet=pet/clientbound/PetMovement version=jms_v185 ida=0x76a534
// packet-audit:verify packet=pet/clientbound/PetMovement version=gms_v84 ida=0x720e70
func TestPetMovementRoundTrip(t *testing.T) {
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			input := NewPetMovement(2001, 0, model.Movement{})
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

// v79 MOVE_PET (cb op 0xAA=170) read order, verified GMS_v79_1_DEVM.exe (port
// 13340): CUserPool::OnUserCommonPacket@0x8c8c79 Decode4(ownerId)@0x8c8c84 →
// CUser::OnPetPacket@0x892474 Decode1(slot)@0x8924b6 → CPet::OnMove@0x690ecb →
// CMovePath::OnMovePacket (opaque movement). Wire = ownerId(4) + slot(1) +
// movement. Codec is version-unconditional; empty model.Movement encodes to
// StartX(2)+StartY(2)+count(1) = 5 zero bytes. Layout byte-identical to v83.
// packet-audit:verify packet=pet/clientbound/PetMovement version=gms_v79 ida=0x690ecb
func TestPetMovementBytesV79(t *testing.T) {
	ctx := test.CreateContext("GMS", 79, 1)
	got := NewPetMovement(0x01020304, 0x05, model.Movement{}).Encode(nil, ctx)(nil)
	want := []byte{
		0x04, 0x03, 0x02, 0x01, // ownerId Decode4@0x8c8c84 (LE)
		0x05,                   // slot Decode1@0x8924b6
		0x00, 0x00,             // movement StartX
		0x00, 0x00,             // movement StartY
		0x00,                   // movement element count = 0
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v79 = % X, want % X", got, want)
	}
}
