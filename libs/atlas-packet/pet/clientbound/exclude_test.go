package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=pet/clientbound/PetExcludeResponse version=gms_v83 ida=0x7061a5
// packet-audit:verify packet=pet/clientbound/PetExcludeResponse version=gms_v87 ida=0x74a17a
// packet-audit:verify packet=pet/clientbound/PetExcludeResponse version=gms_v95 ida=0x6a1510
// packet-audit:verify packet=pet/clientbound/PetExcludeResponse version=jms_v185 ida=0x76be76
// packet-audit:verify packet=pet/clientbound/PetExcludeResponse version=gms_v84 ida=0x722c04
func TestPetExcludeResponse(t *testing.T) {
	input := NewPetExcludeResponse(1234, 0, 999888777, []uint32{2000000, 2000001, 2000002})
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

// v79 PET_EXCEPTION_LIST (cb op 0xAD=173) read order, verified GMS_v79_1_DEVM.exe
// (port 13340): CUserPool::OnUserCommonPacket@0x8c8c79 Decode4(ownerId)@0x8c8c84 →
// CUser::OnPetPacket@0x892474 Decode1(slot)@0x8924b6 →
// CPet::OnLoadExceptionList@0x6928cd: DecodeBuffer(petId,8)@0x6928e8,
// Decode1(count)@0x69291e, count×Decode4(excludeId)@0x69293a. Wire =
// ownerId(4)+slot(1)+petId(8)+count(1)+excludeIds(4 each); identical to v83.
// TestPetExcludeResponseBytesV72 pins the v72 wire = v79 (no version gate).
// IDA GMS_v72.1_U_DEVM.exe @port 13339: CPet::OnLoadExceptionList@0x66da46 reads
// DecodeBuffer(8)(petId)@0x66da61, Decode1(count)@0x66da97, then Decode4(excludeId)
// @0x66dab3 per entry. ownerId + slot are read upstream by CUser::OnPetPacket.
// packet-audit:verify packet=pet/clientbound/PetExcludeResponse version=gms_v72 ida=0x66da46
func TestPetExcludeResponseBytesV72(t *testing.T) {
	ctx := test.CreateContext("GMS", 72, 1)
	got := NewPetExcludeResponse(0x01020304, 0x05, 0x0807060504030201, []uint32{0x11223344, 0x55667788}).Encode(nil, ctx)(nil)
	want := []byte{
		0x04, 0x03, 0x02, 0x01, // ownerId (upstream)
		0x05,                                           // slot (upstream)
		0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, // petId DecodeBuffer(8)@0x66da61 (LE)
		0x02,                   // count Decode1@0x66da97
		0x44, 0x33, 0x22, 0x11, // excludeId[0] Decode4@0x66dab3 (LE)
		0x88, 0x77, 0x66, 0x55, // excludeId[1] Decode4@0x66dab3 (LE)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v72 = % X, want % X", got, want)
	}
}

// packet-audit:verify packet=pet/clientbound/PetExcludeResponse version=gms_v79 ida=0x6928cd
func TestPetExcludeResponseBytesV79(t *testing.T) {
	ctx := test.CreateContext("GMS", 79, 1)
	got := NewPetExcludeResponse(0x01020304, 0x05, 0x0807060504030201, []uint32{0x11223344, 0x55667788}).Encode(nil, ctx)(nil)
	want := []byte{
		0x04, 0x03, 0x02, 0x01, // ownerId Decode4@0x8c8c84 (LE)
		0x05,                                           // slot Decode1@0x8924b6
		0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, // petId DecodeBuffer(8)@0x6928e8 (LE)
		0x02,                   // count Decode1@0x69291e
		0x44, 0x33, 0x22, 0x11, // excludeId[0] Decode4@0x69293a (LE)
		0x88, 0x77, 0x66, 0x55, // excludeId[1] Decode4@0x69293a (LE)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v79 = % X, want % X", got, want)
	}
}
