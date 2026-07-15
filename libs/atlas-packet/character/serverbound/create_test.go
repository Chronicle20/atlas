package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=character/serverbound/CreateCharacter version=gms_v48 ida=0x500545
// packet-audit:verify packet=character/serverbound/CreateCharacter version=gms_v83 ida=0x5f7e7a
// packet-audit:verify packet=character/serverbound/CreateCharacter version=gms_v87 ida=0x62f603
// packet-audit:verify packet=character/serverbound/CreateCharacter version=gms_v95 ida=0x5d7bd0
// packet-audit:verify packet=character/serverbound/CreateCharacter version=gms_v84 ida=0x60cdf0
// packet-audit:verify packet=character/serverbound/CreateCharacter version=jms_v185 ida=0x66e2ab
func TestCreateCharacterRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := CreateCharacter{
				name:             "TestChar",
				jobIndex:         1,
				subJobIndex:      0,
				face:             20000,
				hair:             30000,
				hairColor:        0,
				skinColor:        0,
				topTemplateId:    1040002,
				bottomTemplateId: 1060002,
				shoesTemplateId:  1072001,
				weaponTemplateId: 1302000,
				gender:           0,
				strength:         13,
				dexterity:        4,
				intelligence:     4,
				luck:             4,
			}
			output := CreateCharacter{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Name() != input.Name() {
				t.Errorf("name: got %v, want %v", output.Name(), input.Name())
			}
			if output.Face() != input.Face() {
				t.Errorf("face: got %v, want %v", output.Face(), input.Face())
			}
			if output.Hair() != input.Hair() {
				t.Errorf("hair: got %v, want %v", output.Hair(), input.Hair())
			}
			if output.TopTemplateId() != input.TopTemplateId() {
				t.Errorf("topTemplateId: got %v, want %v", output.TopTemplateId(), input.TopTemplateId())
			}
			if output.BottomTemplateId() != input.BottomTemplateId() {
				t.Errorf("bottomTemplateId: got %v, want %v", output.BottomTemplateId(), input.BottomTemplateId())
			}
			if output.ShoesTemplateId() != input.ShoesTemplateId() {
				t.Errorf("shoesTemplateId: got %v, want %v", output.ShoesTemplateId(), input.ShoesTemplateId())
			}
			if output.WeaponTemplateId() != input.WeaponTemplateId() {
				t.Errorf("weaponTemplateId: got %v, want %v", output.WeaponTemplateId(), input.WeaponTemplateId())
			}
		})
	}
}

// TestCreateCharacterJMSGolden pins the exact jms_v185 wire for CreateCharacter
// against CLogin::SendNewCharPacket @0x66e2ab (non-charSale branch, COutPacket
// 0xB):
//   EncodeStr(name) → Encode4(race/job) → Encode2(subJob) → 6× Encode4(item[i]).
// The 6 ints are the avatar templates (face, hair, top, bottom, shoes, weapon).
// JMS skips hairColor/skinColor and the trailing gender byte (GMS-only).
// TestCreateCharacterV48ByteOutput pins the gms_v48 CREATE_CHAR (op 21). IDA:
// CLogin::SendNewCharPacket = sub_500545 @0x500545 (GMS_v48_1_DEVM.exe) builds
// COutPacket(21) then EncodeStr(name)@0x50058b + 8×Encode4(appearance)@0x5005b5
// (face/hair/hairColor/skinColor/top/bottom/shoes/weapon) + Encode1(gender)@0x5005d9
// + 4×Encode1(str/dex/int/luk)@0x5005f0. Legacy GMS (<73) sends no jobIndex/subJob;
// (<=61) trails the four manually-rolled base stats. Matches the codec gates.
func TestCreateCharacterV48ByteOutput(t *testing.T) {
	ctx := pt.CreateContext("GMS", 48, 1)
	input := CreateCharacter{
		name: "TestChar", jobIndex: 1, subJobIndex: 0,
		face: 20000, hair: 30000, hairColor: 0, skinColor: 0,
		topTemplateId: 1040002, bottomTemplateId: 1060002,
		shoesTemplateId: 1072001, weaponTemplateId: 1302000,
		gender: 0, strength: 13, dexterity: 4, intelligence: 4, luck: 4,
	}
	got := input.Encode(nil, ctx)(nil)
	want := []byte{
		0x08, 0x00, 0x54, 0x65, 0x73, 0x74, 0x43, 0x68, 0x61, 0x72, // EncodeStr "TestChar"
		0x20, 0x4e, 0x00, 0x00, // Encode4 face      = 20000
		0x30, 0x75, 0x00, 0x00, // Encode4 hair      = 30000
		0x00, 0x00, 0x00, 0x00, // Encode4 hairColor = 0
		0x00, 0x00, 0x00, 0x00, // Encode4 skinColor = 0
		0x82, 0xde, 0x0f, 0x00, // Encode4 top       = 1040002
		0xa2, 0x2c, 0x10, 0x00, // Encode4 bottom    = 1060002
		0x81, 0x5b, 0x10, 0x00, // Encode4 shoes     = 1072001
		0xf0, 0xdd, 0x13, 0x00, // Encode4 weapon    = 1302000
		0x00,             // Encode1 gender = 0
		0x0d, 0x04, 0x04, 0x04, // Encode1 str/dex/int/luk = 13/4/4/4
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v48 CreateCharacter wire:\n got %x\nwant %x", got, want)
	}
}

func TestCreateCharacterJMSGolden(t *testing.T) {
	v := pt.Variants[4] // JMS v185
	ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
	input := CreateCharacter{
		name: "TestChar", jobIndex: 1, subJobIndex: 0,
		face: 20000, hair: 30000, hairColor: 0, skinColor: 0,
		topTemplateId: 1040002, bottomTemplateId: 1060002,
		shoesTemplateId: 1072001, weaponTemplateId: 1302000,
		gender: 0, strength: 13, dexterity: 4, intelligence: 4, luck: 4,
	}
	got := input.Encode(nil, ctx)(nil)
	want := []byte{
		0x08, 0x00, 0x54, 0x65, 0x73, 0x74, 0x43, 0x68, 0x61, 0x72, // EncodeStr "TestChar"
		0x01, 0x00, 0x00, 0x00, // Encode4 job/race = 1
		0x00, 0x00, // Encode2 subJob = 0
		0x20, 0x4e, 0x00, 0x00, // Encode4 face   = 20000
		0x30, 0x75, 0x00, 0x00, // Encode4 hair   = 30000
		0x82, 0xde, 0x0f, 0x00, // Encode4 top    = 1040002
		0xa2, 0x2c, 0x10, 0x00, // Encode4 bottom = 1060002
		0x81, 0x5b, 0x10, 0x00, // Encode4 shoes  = 1072001
		0xf0, 0xdd, 0x13, 0x00, // Encode4 weapon = 1302000
	}
	if !bytes.Equal(got, want) {
		t.Errorf("jms CreateCharacter wire:\n got %x\nwant %x", got, want)
	}
}
