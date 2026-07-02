package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=pet/serverbound/PetCommand version=gms_v83 ida=0x704d5d
// packet-audit:verify packet=pet/serverbound/PetCommand version=gms_v87 ida=0x748a35
// packet-audit:verify packet=pet/serverbound/PetCommand version=gms_v95 ida=0x6a3cc0
// packet-audit:verify packet=pet/serverbound/PetCommand version=jms_v185 ida=0x76abe0
// packet-audit:verify packet=pet/serverbound/PetCommand version=gms_v84 ida=0x7214bf
func TestCommandRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := Command{petId: 12345, byName: true, command: 3}
			output := Command{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.PetId() != input.PetId() {
				t.Errorf("petId: got %v, want %v", output.PetId(), input.PetId())
			}
			if output.ByName() != input.ByName() {
				t.Errorf("byName: got %v, want %v", output.ByName(), input.ByName())
			}
			if output.Command() != input.Command() {
				t.Errorf("command: got %v, want %v", output.Command(), input.Command())
			}
		})
	}
}

// v79 PET_COMMAND (sb op 165=0xA5) send order, verified GMS_v79_1_DEVM.exe (port
// 13340): sub_6914DB — COutPacket(165)@0x69171c, EncodeBuffer(petId,8)@0x691731,
// Encode1(byName/v35)@0x69173c, Encode1(command/v38)@0x691747. Wire =
// petId(8)+byName(1)+command(1); byte-identical to v83.
// TestCommandBytesV72 pins the v72 wire = v79 (no version gate). IDA
// GMS_v72.1_U_DEVM.exe @port 13339: CPet::ParseCommand@0x66c67b send block builds
// COutPacket(163)@0x66c8b3, EncodeBuffer(petId,8)@0x66c8c8, Encode1(byName)@0x66c8d3,
// Encode1(command)@0x66c8de.
// packet-audit:verify packet=pet/serverbound/PetCommand version=gms_v72 ida=0x66c67b
func TestCommandBytesV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	got := Command{petId: 0x0102030405060708, byName: true, command: 0x09}.Encode(nil, ctx)(nil)
	want := []byte{
		0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01, // petId EncodeBuffer(8)@0x66c8c8 (LE)
		0x01, // byName Encode1@0x66c8d3
		0x09, // command Encode1@0x66c8de
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v72 = % X, want % X", got, want)
	}
}

// TestCommandBytesV61 pins the v61 wire = v72 (no version gate). IDA
// GMS_v61.1_U_DEVM.exe @port 13338: CPet::ParseCommand sub_613B18@0x613b18 send
// block builds COutPacket(140)@0x613d51, EncodeBuffer(petId,8)@0x613d66,
// Encode1(byName)@0x613d71, Encode1(command)@0x613d7c. v72 op163 (Δ-23).
// packet-audit:verify packet=pet/serverbound/PetCommand version=gms_v61 ida=0x613b18
func TestCommandBytesV61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	got := Command{petId: 0x0102030405060708, byName: true, command: 0x09}.Encode(nil, ctx)(nil)
	want := []byte{
		0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01, // petId EncodeBuffer(8)@0x613d66 (LE)
		0x01, // byName Encode1@0x613d71
		0x09, // command Encode1@0x613d7c
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v61 = % X, want % X", got, want)
	}
}

// packet-audit:verify packet=pet/serverbound/PetCommand version=gms_v79 ida=0x6914db
func TestCommandBytesV79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	got := Command{petId: 0x0102030405060708, byName: true, command: 0x09}.Encode(nil, ctx)(nil)
	want := []byte{
		0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01, // petId EncodeBuffer(8)@0x691731 (LE)
		0x01, // byName Encode1@0x69173c
		0x09, // command Encode1@0x691747
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v79 = % X, want % X", got, want)
	}
}
