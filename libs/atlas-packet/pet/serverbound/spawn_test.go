package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=pet/serverbound/PetSpawn version=gms_v87 ida=0xabbb70
// packet-audit:verify packet=pet/serverbound/PetSpawn version=gms_v95 ida=0x9f6980
// packet-audit:verify packet=pet/serverbound/PetSpawn version=jms_v185 ida=0xb0b40b
func TestSpawnRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := Spawn{updateTime: 100, slot: -5, lead: true}
			output := Spawn{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.UpdateTime() != input.UpdateTime() {
				t.Errorf("updateTime: got %v, want %v", output.UpdateTime(), input.UpdateTime())
			}
			if output.Slot() != input.Slot() {
				t.Errorf("slot: got %v, want %v", output.Slot(), input.Slot())
			}
			if output.Lead() != input.Lead() {
				t.Errorf("lead: got %v, want %v", output.Lead(), input.Lead())
			}
		})
	}
}

// v79 SPAWN_PET (sb op 96=0x60) send order, verified GMS_v79_1_DEVM.exe (port
// 13340): CWvsContext::SendActivatePetRequest (sub_96E251) — COutPacket(96)@0x96e550,
// Encode4(updateTime)@0x96e56a, Encode2(slot)@0x96e575, Encode1(lead)@0x96e580.
// Wire = updateTime(4)+slot(2)+lead(1); byte-identical to v83.
// TestSpawnBytesV72 pins the v72 wire = v79 (no version gate). IDA
// GMS_v72.1_U_DEVM.exe @port 13339: CWvsContext::SendActivatePetRequest@0x91c241
// op-97 send block builds COutPacket(97)@0x91c4f9 then Encode4(updateTime)@0x91c513,
// Encode2(slot)@0x91c51e, Encode1(lead)@0x91c529.
// packet-audit:verify packet=pet/serverbound/PetSpawn version=gms_v72 ida=0x91c241
func TestSpawnBytesV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	got := Spawn{updateTime: 0x01020304, slot: 0x0506, lead: true}.Encode(nil, ctx)(nil)
	want := []byte{
		0x04, 0x03, 0x02, 0x01, // updateTime Encode4@0x91c513 (LE)
		0x06, 0x05, // slot Encode2@0x91c51e (LE)
		0x01, // lead Encode1@0x91c529
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v72 = % X, want % X", got, want)
	}
}

// packet-audit:verify packet=pet/serverbound/PetSpawn version=gms_v79 ida=0x96e251
func TestSpawnBytesV79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	got := Spawn{updateTime: 0x01020304, slot: 0x0506, lead: true}.Encode(nil, ctx)(nil)
	want := []byte{
		0x04, 0x03, 0x02, 0x01, // updateTime Encode4@0x96e56a (LE)
		0x06, 0x05, // slot Encode2@0x96e575 (LE)
		0x01, // lead Encode1@0x96e580
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v79 = % X, want % X", got, want)
	}
}
