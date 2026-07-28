package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=pet/serverbound/PetDropPickUp version=gms_v87 ida=0x749be8
// packet-audit:verify packet=pet/serverbound/PetDropPickUp version=gms_v95 ida=0x6a0820
// packet-audit:verify packet=pet/serverbound/PetDropPickUp version=gms_v83 ida=0x705c7c
// packet-audit:verify packet=pet/serverbound/PetDropPickUp version=jms_v185 ida=0x76bcc6
// packet-audit:verify packet=pet/serverbound/PetDropPickUp version=gms_v84 ida=0x722672
func TestDropPickUpRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			// Use dropId=14 (14%13!=0) to exercise the extended fields path
			input := DropPickUp{petId: 12345, fieldKey: 1, updateTime: 100, x: 50, y: -30, dropId: 14, crc: 999, bPickupOthers: true, bSweepForDrop: false, bLongRange: true, ownerX: 10, ownerY: 20, posCrc: 111, rectCrc: 222}
			output := DropPickUp{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			// GMS v48 and older drop the leading petId (single-pet); only assert it
			// on versions that carry the id (GMS v61+ or JMS).
			if (v.Region != "GMS" || v.MajorVersion >= 61) && output.PetId() != input.PetId() {
				t.Errorf("petId: got %v, want %v", output.PetId(), input.PetId())
			}
			if output.DropId() != input.DropId() {
				t.Errorf("dropId: got %v, want %v", output.DropId(), input.DropId())
			}
			if output.BPickupOthers() != input.BPickupOthers() {
				t.Errorf("bPickupOthers: got %v, want %v", output.BPickupOthers(), input.BPickupOthers())
			}
		})
	}
}

func TestDropPickUpDivisibleByThirteenRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			// Use dropId=26 (26%13==0) to exercise the non-extended path
			input := DropPickUp{petId: 12345, fieldKey: 1, updateTime: 100, x: 50, y: -30, dropId: 26, crc: 999, bPickupOthers: false, bSweepForDrop: true, bLongRange: false}
			output := DropPickUp{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.DropId() != input.DropId() {
				t.Errorf("dropId: got %v, want %v", output.DropId(), input.DropId())
			}
			if output.BSweepForDrop() != input.BSweepForDrop() {
				t.Errorf("bSweepForDrop: got %v, want %v", output.BSweepForDrop(), input.BSweepForDrop())
			}
		})
	}
}

// v79 PET_LOOT (sb op 166=0xA6) send order, verified GMS_v79_1_DEVM.exe (port
// 13340): sub_6923AF — COutPacket(166)@0x6923ea, EncodeBuffer(petId,8)@0x6923ff,
// Encode1(fieldKey)@0x692418, Encode4(updateTime)@0x692426, Encode2(x)@0x692437,
// Encode2(y)@0x692446, Encode4(dropId)@0x692451, Encode1(bPickupOthers)@0x69246e,
// Encode1(bSweepForDrop)@0x69248b, Encode1(bLongRange)@0x6924a8.
//
// DIVERGENCE vs v83+: v79 has NO crc field. v83 CPet::SendDropPickUpRequest@0x705c7c
// (v83 IDB, port 13342) adds a second Encode4(a5=crc)@0x705d29 after
// Encode4(a4=dropId)@0x705d1e; v79 goes straight from dropId to the bool bytes.
// The codec gates crc to (GMS>=83 || JMS), so v79 omits it. The v87+ owner block
// is also absent (MajorAtLeast(87) false on v79).
// TestDropPickUpBytesV72 pins the v72 wire = v79 (no crc, GMS<83; no v87+ block).
// IDA GMS_v72.1_U_DEVM.exe @port 13339: CPet::SendDropPickUpRequest@0x66d52b builds
// COutPacket(164)@0x66d566, EncodeBuffer(petId,8)@0x66d57b, Encode1(fieldKey)@0x66d594,
// Encode4(updateTime)@0x66d5a2, Encode2(x)@0x66d5b3, Encode2(y)@0x66d5c2,
// Encode4(dropId)@0x66d5cd, then STRAIGHT to Encode1(bPickupOthers)@0x66d5ea,
// Encode1(bSweepForDrop)@0x66d607, Encode1(bLongRange)@0x66d624 — no crc int.
// packet-audit:verify packet=pet/serverbound/PetDropPickUp version=gms_v72 ida=0x66d52b
func TestDropPickUpBytesV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	in := DropPickUp{petId: 0x0102030405060708, fieldKey: 0x09, updateTime: 0x0A0B0C0D, x: 0x1011, y: 0x1213, dropId: 0x14, crc: 0x99999999, bPickupOthers: true, bSweepForDrop: false, bLongRange: true}
	got := in.Encode(nil, ctx)(nil)
	want := []byte{
		0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01, // petId EncodeBuffer(8)@0x66d57b (LE)
		0x09,                   // fieldKey Encode1@0x66d594
		0x0D, 0x0C, 0x0B, 0x0A, // updateTime Encode4@0x66d5a2 (LE)
		0x11, 0x10, // x Encode2@0x66d5b3 (LE)
		0x13, 0x12, // y Encode2@0x66d5c2 (LE)
		0x14, 0x00, 0x00, 0x00, // dropId Encode4@0x66d5cd (LE) — NO crc follows on v72
		0x01, // bPickupOthers Encode1@0x66d5ea
		0x00, // bSweepForDrop Encode1@0x66d607
		0x01, // bLongRange Encode1@0x66d624
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v72 = % X, want % X", got, want)
	}
}

// TestDropPickUpBytesV61 pins the v61 wire = v72 (no crc, GMS<83; no v87+ block).
// IDA GMS_v61.1_U_DEVM.exe @port 13338: CPet::SendDropPickUpRequest sub_6149DB@0x6149db
// builds COutPacket(141)@0x6149f6, EncodeBuffer(petId,8)@0x614a0b, Encode1(fieldKey)
// @0x614a24, Encode4(updateTime)@0x614a32, Encode2(x)@0x614a43, Encode2(y)@0x614a52,
// Encode4(dropId)@0x614a5d, then STRAIGHT to Encode1(bPickupOthers)@0x614a7a,
// Encode1(bSweepForDrop)@0x614a97, Encode1(bLongRange)@0x614ab4 — no crc int (v61<83).
// v72 op164 (Δ-23).
// packet-audit:verify packet=pet/serverbound/PetDropPickUp version=gms_v61 ida=0x6149db
func TestDropPickUpBytesV61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	in := DropPickUp{petId: 0x0102030405060708, fieldKey: 0x09, updateTime: 0x0A0B0C0D, x: 0x1011, y: 0x1213, dropId: 0x14, crc: 0x99999999, bPickupOthers: true, bSweepForDrop: false, bLongRange: true}
	got := in.Encode(nil, ctx)(nil)
	want := []byte{
		0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01, // petId EncodeBuffer(8)@0x614a0b (LE)
		0x09,                   // fieldKey Encode1@0x614a24
		0x0D, 0x0C, 0x0B, 0x0A, // updateTime Encode4@0x614a32 (LE)
		0x11, 0x10, // x Encode2@0x614a43 (LE)
		0x13, 0x12, // y Encode2@0x614a52 (LE)
		0x14, 0x00, 0x00, 0x00, // dropId Encode4@0x614a5d (LE) — NO crc follows on v61
		0x01, // bPickupOthers Encode1@0x614a7a
		0x00, // bSweepForDrop Encode1@0x614a97
		0x01, // bLongRange Encode1@0x614ab4
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v61 = % X, want % X", got, want)
	}
}

// packet-audit:verify packet=pet/serverbound/PetDropPickUp version=gms_v79 ida=0x6923af
func TestDropPickUpBytesV79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	in := DropPickUp{petId: 0x0102030405060708, fieldKey: 0x09, updateTime: 0x0A0B0C0D, x: 0x1011, y: 0x1213, dropId: 0x14, crc: 0x99999999, bPickupOthers: true, bSweepForDrop: false, bLongRange: true}
	got := in.Encode(nil, ctx)(nil)
	want := []byte{
		0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01, // petId EncodeBuffer(8)@0x6923ff (LE)
		0x09,                   // fieldKey Encode1@0x692418
		0x0D, 0x0C, 0x0B, 0x0A, // updateTime Encode4@0x692426 (LE)
		0x11, 0x10, // x Encode2@0x692437 (LE)
		0x13, 0x12, // y Encode2@0x692446 (LE)
		0x14, 0x00, 0x00, 0x00, // dropId Encode4@0x692451 (LE) — NO crc follows on v79
		0x01, // bPickupOthers Encode1@0x69246e
		0x00, // bSweepForDrop Encode1@0x69248b
		0x01, // bLongRange Encode1@0x6924a8
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v79 = % X, want % X", got, want)
	}
	// crc gate cross-check: the same fixture under v83 carries the extra 4-byte crc.
	b83 := in.Encode(nil, pt.CreateContext("GMS", 83, 1))(nil)
	if len(b83)-len(got) != 4 {
		t.Fatalf("v83 len %d vs v79 len %d: want v79 to omit the 4-byte crc", len(b83), len(got))
	}
}

// TestDropPickUpBytesV48 pins the v48 PET_LOOT (sb op 116 / 0x74) send. IDA
// GMS_v48_1_DEVM.exe @port 13337: sub_58ED98@0x58edb0 builds COutPacket(116),
// Encode1(fieldKey)@0x58edcd, Encode4(updateTime)@0x58eddb, Encode2(x)@0x58edec,
// Encode2(y)@0x58edfb, Encode4(dropId)@0x58ee06, Encode1(bPickupOthers)@0x58ee23,
// Encode1(bSweepForDrop)@0x58ee40, Encode1(bLongRange)@0x58ee5d — NO leading
// EncodeBuffer(petId,8) (single-pet), NO crc int (v48<83). v61 op141 carries petId.
// packet-audit:verify packet=pet/serverbound/PetDropPickUp version=gms_v48 ida=0x58ed98
func TestDropPickUpBytesV48(t *testing.T) {
	ctx := pt.CreateContext("GMS", 48, 1)
	in := DropPickUp{petId: 0x0102030405060708, fieldKey: 0x09, updateTime: 0x0A0B0C0D, x: 0x1011, y: 0x1213, dropId: 0x14, crc: 0x99999999, bPickupOthers: true, bSweepForDrop: false, bLongRange: true}
	got := in.Encode(nil, ctx)(nil)
	want := []byte{
		// NO petId on v48 (single-pet)
		0x09,                   // fieldKey Encode1@0x58edcd
		0x0D, 0x0C, 0x0B, 0x0A, // updateTime Encode4@0x58eddb (LE)
		0x11, 0x10, // x Encode2@0x58edec (LE)
		0x13, 0x12, // y Encode2@0x58edfb (LE)
		0x14, 0x00, 0x00, 0x00, // dropId Encode4@0x58ee06 (LE) — NO crc on v48
		0x01, // bPickupOthers Encode1@0x58ee23
		0x00, // bSweepForDrop Encode1@0x58ee40
		0x01, // bLongRange Encode1@0x58ee5d
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v48 = % X, want % X", got, want)
	}
}
