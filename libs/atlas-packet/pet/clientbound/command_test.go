package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=pet/clientbound/PetCommandResponse version=gms_v83 ida=0x7048ab
// packet-audit:verify packet=pet/clientbound/PetCommandResponse version=gms_v87 ida=0x74858a
// packet-audit:verify packet=pet/clientbound/PetCommandResponse version=gms_v95 ida=0x6a3930
// packet-audit:verify packet=pet/clientbound/PetCommandResponse version=jms_v185 ida=0x76a6ab
// packet-audit:verify packet=pet/clientbound/PetCommandResponse version=gms_v84 ida=0x720fd0
func TestPetCommandResponse(t *testing.T) {
	input := NewPetCommandResponse(1234, 0, 3, true, false)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestPetFoodResponse(t *testing.T) {
	input := NewPetFoodResponse(1234, 1, 5, false, true)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

// v79 PET_COMMAND (cb op 0xAE=174) read order, verified GMS_v79_1_DEVM.exe (port
// 13340): CUserPool::OnUserCommonPacket@0x8c8c79 Decode4(ownerId)@0x8c8c84 →
// CUser::OnPetPacket@0x892474 Decode1(slot)@0x8924b6 →
// CPet::OnActionCommand@0x691029: Decode1(mode/v23)@0x69105f; mode==0 path reads
// Decode1(animation)@0x691088, Decode1(success)@0x6910a1, then Decode1(balloon)@0x6911d4
// (the static report's "trailing padding" note is wrong — the balloon byte IS read).
// Wire = ownerId(4)+slot(1)+mode(1)+animation(1)+success(1)+balloon(1); identical to v83.
// TestPetCommandResponseBytesV72 pins the v72 wire = v79 (no version gate).
// IDA GMS_v72.1_U_DEVM.exe @port 13339: CPet::OnActionCommand@0x66c1de reads
// Decode1(mode)@0x66c214; for mode 0 it then reads Decode1(animation)@0x66c23a,
// Decode1(success)@0x66c253, and Decode1(balloon)@0x66c37d (mode/animation/
// success/balloon = 4 bytes). ownerId + slot are read upstream by CUser::OnPetPacket.
// packet-audit:verify packet=pet/clientbound/PetCommandResponse version=gms_v72 ida=0x66c1de
func TestPetCommandResponseBytesV72(t *testing.T) {
	ctx := test.CreateContext("GMS", 72, 1)
	got := NewPetCommandResponse(0x01020304, 0x05, 0x07, true, false).Encode(nil, ctx)(nil)
	want := []byte{
		0x04, 0x03, 0x02, 0x01, // ownerId (upstream)
		0x05, // slot (upstream)
		0x00, // mode Decode1@0x66c214 (NewPetCommandResponse => mode 0)
		0x07, // animation Decode1@0x66c23a
		0x01, // success Decode1@0x66c253
		0x00, // balloon Decode1@0x66c37d
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v72 = % X, want % X", got, want)
	}
}

// packet-audit:verify packet=pet/clientbound/PetCommandResponse version=gms_v79 ida=0x691029
func TestPetCommandResponseBytesV79(t *testing.T) {
	ctx := test.CreateContext("GMS", 79, 1)
	got := NewPetCommandResponse(0x01020304, 0x05, 0x07, true, false).Encode(nil, ctx)(nil)
	want := []byte{
		0x04, 0x03, 0x02, 0x01, // ownerId Decode4@0x8c8c84 (LE)
		0x05, // slot Decode1@0x8924b6
		0x00, // mode Decode1@0x69105f (NewPetCommandResponse => mode 0)
		0x07, // animation Decode1@0x691088
		0x01, // success Decode1@0x6910a1
		0x00, // balloon Decode1@0x6911d4
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v79 = % X, want % X", got, want)
	}
}
