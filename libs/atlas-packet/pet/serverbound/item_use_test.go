package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=pet/serverbound/PetItemUse version=gms_v83 ida=0xa0955c
// packet-audit:verify packet=pet/serverbound/PetItemUse version=gms_v87 ida=0xa9ee08
// packet-audit:verify packet=pet/serverbound/PetItemUse version=gms_v95 ida=0x9de400
// packet-audit:verify packet=pet/serverbound/PetItemUse version=jms_v185 ida=0xaee1d4
// packet-audit:verify packet=pet/serverbound/PetItemUse version=gms_v84 ida=0xa5393e
func TestItemUseRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ItemUse{petId: 12345, buffSkill: true, updateTime: 100, source: 5, itemId: 2000001}
			output := ItemUse{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			// GMS v48 omits the leading petId (single-pet client, encoder @0x70dc8d
			// has no EncodeBuffer(petSN,8)); only assert it on versions that carry
			// the id (GMS v61+ or JMS).
			if (v.Region != "GMS" || v.MajorVersion >= 61) && output.PetId() != input.PetId() {
				t.Errorf("petId: got %v, want %v", output.PetId(), input.PetId())
			}
			if output.BuffSkill() != input.BuffSkill() {
				t.Errorf("buffSkill: got %v, want %v", output.BuffSkill(), input.BuffSkill())
			}
			if output.UpdateTime() != input.UpdateTime() {
				t.Errorf("updateTime: got %v, want %v", output.UpdateTime(), input.UpdateTime())
			}
			if output.Source() != input.Source() {
				t.Errorf("source: got %v, want %v", output.Source(), input.Source())
			}
			if output.ItemId() != input.ItemId() {
				t.Errorf("itemId: got %v, want %v", output.ItemId(), input.ItemId())
			}
		})
	}
}

// TestItemUseBytesV61 pins the v61 PET_AUTO_POT (sb op 142=0x8E) send order.
// IDA GMS_v61.1_U_DEVM.exe (session 965202bf):
// CWvsContext::SendStatChangeItemUseRequestByPetQ@0x831ab9 —
// COutPacket(142)@0x831b16, EncodeBuffer(petSN,8)@0x831b27,
// Encode1(a7=buffSkill)@0x831b32, Encode4(v10=updateTime)@0x831b40,
// Encode2(a4=source)@0x831b4b, Encode4(a5=itemId)@0x831b56. Wire =
// petId(8)+buffSkill(1)+updateTime(4)+source(2)+itemId(4); byte-identical to
// gms_v83.
// packet-audit:verify packet=pet/serverbound/PetItemUse version=gms_v61 ida=0x831ab9
func TestItemUseBytesV61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	got := ItemUse{petId: 0x0102030405060708, buffSkill: true, updateTime: 100, source: 5, itemId: 2000001}.Encode(nil, ctx)(nil)
	want := []byte{
		0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01, // petId EncodeBuffer(8)@0x831b27 (LE)
		0x01,                   // buffSkill Encode1@0x831b32
		0x64, 0x00, 0x00, 0x00, // updateTime Encode4@0x831b40 (100 LE)
		0x05, 0x00, // source Encode2@0x831b4b (5 LE)
		0x81, 0x84, 0x1E, 0x00, // itemId Encode4@0x831b56 (2000001 LE)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v61 = % X, want % X", got, want)
	}
}

// TestItemUseBytesV72 pins the v72 wire = v61 (no version gate). IDA
// GMS_v72.1_U_DEVM.exe (session 90e36cb0):
// CWvsContext::SendStatChangeItemUseRequestByPetQ@0x903f8b —
// COutPacket(165)@0x903fe5, EncodeBuffer(petSN,8)@0x903ff7,
// Encode1(a7=buffSkill)@0x904002, Encode4(v10=updateTime)@0x904010,
// Encode2(a4=source)@0x90401b, Encode4(a5=itemId)@0x904026.
// packet-audit:verify packet=pet/serverbound/PetItemUse version=gms_v72 ida=0x903f8b
func TestItemUseBytesV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	got := ItemUse{petId: 0x0102030405060708, buffSkill: true, updateTime: 100, source: 5, itemId: 2000001}.Encode(nil, ctx)(nil)
	want := []byte{
		0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01, // petId EncodeBuffer(8)@0x903ff7 (LE)
		0x01,                   // buffSkill Encode1@0x904002
		0x64, 0x00, 0x00, 0x00, // updateTime Encode4@0x904010 (100 LE)
		0x05, 0x00, // source Encode2@0x90401b (5 LE)
		0x81, 0x84, 0x1E, 0x00, // itemId Encode4@0x904026 (2000001 LE)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v72 = % X, want % X", got, want)
	}
}

// TestItemUseBytesV79 pins the v79 wire = v61/v72 (no version gate). IDA
// GMS_v79_1_DEVM.exe (session 9a7d3642):
// CWvsContext::SendStatChangeItemUseRequestByPetQ@0x9552d0 —
// COutPacket(167)@0x95532a, EncodeBuffer(petSN,8)@0x95533c,
// Encode1(a7=buffSkill)@0x955347, Encode4(v10=updateTime)@0x955355,
// Encode2(a4=source)@0x955360, Encode4(a5=itemId)@0x95536b.
// packet-audit:verify packet=pet/serverbound/PetItemUse version=gms_v79 ida=0x9552d0
func TestItemUseBytesV79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	got := ItemUse{petId: 0x0102030405060708, buffSkill: true, updateTime: 100, source: 5, itemId: 2000001}.Encode(nil, ctx)(nil)
	want := []byte{
		0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01, // petId EncodeBuffer(8)@0x95533c (LE)
		0x01,                   // buffSkill Encode1@0x955347
		0x64, 0x00, 0x00, 0x00, // updateTime Encode4@0x955355 (100 LE)
		0x05, 0x00, // source Encode2@0x955360 (5 LE)
		0x81, 0x84, 0x1E, 0x00, // itemId Encode4@0x95536b (2000001 LE)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v79 = % X, want % X", got, want)
	}
}

// TestItemUseBytesV92 pins the v92 PET_AUTO_POT (sb op 200=0xC8) send order.
// IDA GMS_v92_1_DEVM.exe (session acdfccff):
// CWvsContext::SendStatChangeItemUseRequestByPetQ@0x9b3a00 —
// COutPacket(0xC8)@0x9b3abe (200, matches registry opcode 200),
// EncodeBuffer(&Src,8)@0x9b3ad6 (petId), Encode1(a7=buffSkill)@0x9b3ae4,
// Encode4(v14=updateTime via sub_936E80)@0x9b3af3, Encode2(a4=source)@0x9b3b01,
// Encode4(v10=itemId/Args)@0x9b3b0b. Wire = petId(8)+buffSkill(1)+
// updateTime(4)+source(2)+itemId(4); byte-identical to gms_v61/v72/v79/v83.
// packet-audit:verify packet=pet/serverbound/PetItemUse version=gms_v92 ida=0x9b3a00
func TestItemUseBytesV92(t *testing.T) {
	ctx := pt.CreateContext("GMS", 92, 1)
	got := ItemUse{petId: 0x0102030405060708, buffSkill: true, updateTime: 100, source: 5, itemId: 2000001}.Encode(nil, ctx)(nil)
	want := []byte{
		0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01, // petId EncodeBuffer(8)@0x9b3ad6 (LE)
		0x01,                   // buffSkill Encode1@0x9b3ae4
		0x64, 0x00, 0x00, 0x00, // updateTime Encode4@0x9b3af3 (100 LE)
		0x05, 0x00, // source Encode2@0x9b3b01 (5 LE)
		0x81, 0x84, 0x1E, 0x00, // itemId Encode4@0x9b3b0b (2000001 LE)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v92 = % X, want % X", got, want)
	}
}

// TestItemUseBytesV48 pins the v48 PET_AUTO_POT (sb op 117=0x75) send — NO
// leading petSN (v48 is single-pet). IDA GMS_v48_1_DEVM.exe (session
// 0bb5f11a): sub_70DC8D@0x70dc8d (named
// CWvsContext::SendStatChangeItemUseRequestByPetQ) — COutPacket(117)@0x70dcc6,
// Encode1(a4=buffSkill)@0x70dcd5, Encode4(v7=updateTime)@0x70dce3,
// Encode2(a2=source)@0x70dcee, Encode4(a3=itemId)@0x70dcf9. NO
// EncodeBuffer(petSN,8) anywhere in this function — v61 op142 is the first to
// carry it.
// packet-audit:verify packet=pet/serverbound/PetItemUse version=gms_v48 ida=0x70dc8d
func TestItemUseBytesV48(t *testing.T) {
	ctx := pt.CreateContext("GMS", 48, 1)
	got := ItemUse{petId: 0x0102030405060708, buffSkill: true, updateTime: 100, source: 5, itemId: 2000001}.Encode(nil, ctx)(nil)
	want := []byte{
		// NO petId on v48 (single-pet)
		0x01,                   // buffSkill Encode1@0x70dcd5
		0x64, 0x00, 0x00, 0x00, // updateTime Encode4@0x70dce3 (100 LE)
		0x05, 0x00, // source Encode2@0x70dcee (5 LE)
		0x81, 0x84, 0x1E, 0x00, // itemId Encode4@0x70dcf9 (2000001 LE)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v48 = % X, want % X", got, want)
	}
}
