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
			if output.PetId() != input.PetId() {
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
