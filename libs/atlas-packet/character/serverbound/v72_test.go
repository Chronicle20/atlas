package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// v72 character serverbound byte fixtures (GMS_v72.1_U_DEVM.exe, port 13339).

// CheckName v72 byte-fixture.
//
// Client send — CLogin::SendCheckDuplicateIDPacket sub_46BB81 @0x46BB81:
//   COutPacket(21) @0x46bbe7 then EncodeStr(name) @0x46bc04. Single name string.
//   == v79.
//
// packet-audit:verify packet=character/serverbound/CheckName version=gms_v72 ida=0x46bb81
func TestCheckNameByteOutputV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	got := CheckName{name: "TestChar"}.Encode(nil, ctx)(nil)
	want := []byte{
		0x08, 0x00, // EncodeStr length = 8       /*0x46bc04*/
		0x54, 0x65, 0x73, 0x74, 0x43, 0x68, 0x61, 0x72, // "TestChar"
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v72 CheckName wire: got %x want %x", got, want)
	}
}

// CreateCharacter v72 byte-fixture.
//
// Client send — CLogin::SendNewCharPacket sub_5B219A @0x5B219A:
//   COutPacket(22)                                                  /*0x5b2206*/
//   EncodeStr(name)                                                 /*0x5b221f*/
//   loop 8x Encode4 (face, hair, hairColor, skinColor, top, bottom, shoes, weapon) /*0x5b2230*/
//   Encode1(gender)                                                 /*0x5b2247*/
//
// LEGACY DIVERGENCE vs v79: v72 (< 73) writes NO jobIndex int (the do/while loop
// runs exactly 8 times; there is no separate Encode4 before it). create.go already
// gates the jobIndex on GMS>=73, so the v72 codec is byte-correct: name + 8 ints +
// gender, with no jobIndex.
//
// packet-audit:verify packet=character/serverbound/CreateCharacter version=gms_v72 ida=0x5b219a
func TestCreateCharacterByteOutputV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	input := CreateCharacter{
		name:             "TestChar",
		jobIndex:         1, // NOT written for v72 (<73)
		subJobIndex:      0,
		face:             20000, // 0x4E20
		hair:             30000, // 0x7530
		hairColor:        0,
		skinColor:        0,
		topTemplateId:    1040002, // 0x0FDE82
		bottomTemplateId: 1060002, // 0x102CA2
		shoesTemplateId:  1072001, // 0x105D81
		weaponTemplateId: 1302000, // 0x13DDF0
		gender:           0,
	}
	got := input.Encode(nil, ctx)(nil)
	want := []byte{
		0x08, 0x00, // name length = 8            /*0x5b221f*/
		0x54, 0x65, 0x73, 0x74, 0x43, 0x68, 0x61, 0x72, // "TestChar"
		// NO jobIndex int (v72 < 73)                /*(v79 @0x5cd037)*/
		0x20, 0x4e, 0x00, 0x00, // face = 20000   /*0x5b2230 loop*/
		0x30, 0x75, 0x00, 0x00, // hair = 30000
		0x00, 0x00, 0x00, 0x00, // hairColor = 0
		0x00, 0x00, 0x00, 0x00, // skinColor = 0
		0x82, 0xde, 0x0f, 0x00, // top = 1040002
		0xa2, 0x2c, 0x10, 0x00, // bottom = 1060002
		0x81, 0x5b, 0x10, 0x00, // shoes = 1072001
		0xf0, 0xdd, 0x13, 0x00, // weapon = 1302000
		0x00, // gender                           /*0x5b2247*/
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v72 CreateCharacter wire:\n got %x\nwant %x", got, want)
	}
}

// DeleteCharacter v72 byte-fixture.
//
// Client send — CLogin::SendDeleteCharPacket sub_5B206B @0x5B206B (the login
// delete-character sender; the registry's stale 0x7169BE points at an unrelated
// dialog sender. Verified by body: prompts DOB via sub_5C728E, sends [dob][charId]
// and deselects the slot):
//   COutPacket(24)                                                  /*0x5b2110*/
//   Encode4(dob)   // date-of-birth security value                  /*0x5b211e*/
//   Encode4(characterId)                                            /*0x5b213b*/
//
// GROUND-TRUTH OPCODE: COutPacket(24), NOT 23. The CSV has no v72 column; the
// registry seeded op 23 from the v83 column, but the v72 client sends opcode 24
// (matching the v92/v95 CSV column). registry gms_v72.yaml + template_gms_72_1.json
// corrected to op 0x18 (24). Body is unchanged: [int dob][int characterId].
//
// packet-audit:verify packet=character/serverbound/DeleteCharacter version=gms_v72 ida=0x5b206b
func TestDeleteCharacterByteOutputV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	input := DeleteCharacter{dob: 19900101, characterId: 12345}
	got := input.Encode(nil, ctx)(nil)
	want := []byte{
		0xc5, 0xa6, 0x2f, 0x01, // dob = 19900101 (Encode4)   /*0x5b211e*/
		0x39, 0x30, 0x00, 0x00, // characterId = 12345         /*0x5b213b*/
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v72 DeleteCharacter wire: got %x want %x", got, want)
	}
}

// DistributeAp v72 byte-fixture — DISTRIBUTE_AP, op 86 (0x56).
//
// Client send — CWvsContext::SendAbilityUpRequest(DWORD) sub_91BBAD @0x91BBAD:
//   COutPacket(86)                                                  /*0x91bc86*/
//   Encode4(update_time)                                            /*0x91bc97*/
//   Encode4(dwFlag)  // ability-up bitmask (a2)                     /*0x91bca2*/
//
// Body = updateTime(4) + dwFlag(4). Version-invariant vs v79.
//
// packet-audit:verify packet=character/serverbound/DistributeAp version=gms_v72 ida=0x91bbad
func TestDistributeApByteOutputV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	input := DistributeAp{updateTime: 100, dwFlag: 0x20}
	got := pt.Encode(t, ctx, input.Encode, nil)
	want := []byte{
		0x64, 0x00, 0x00, 0x00, // updateTime 100 (Encode4)  /*0x91bc97*/
		0x20, 0x00, 0x00, 0x00, // dwFlag 0x20 (Encode4)     /*0x91bca2*/
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v72 DistributeAp wire:\n got %x\nwant %x", got, want)
	}
}

// AutoDistributeAp v72 byte-fixture — AUTO_DISTRIBUTE_AP, op 87 (0x57).
//
// Client send — CWvsContext::SendAbilityUpRequest(ZArray<StatPair>*) sub_91BCE8 @0x91BCE8:
//   COutPacket(87)                                                  /*0x91be17*/
//   Encode4(update_time)                                            /*0x91be29*/
//   Encode4(count)  // *(a2-4) array length                         /*0x91be3b*/
//   loop i: Encode4(flag[i]) /*0x91be53*/, Encode4(value[i]) /*0x91be61*/
//
// Body = updateTime(4) + count(4) + count×(flag(4)+value(4)). == v79.
//
// packet-audit:verify packet=character/serverbound/AutoDistributeAp version=gms_v72 ida=0x91bce8
func TestAutoDistributeApByteOutputV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	input := AutoDistributeAp{
		updateTime: 100,
		nValue:     2,
		distributes: []DistributeEntry{
			{Flag: 0x40, Value: 1},
			{Flag: 0x80, Value: 2},
		},
	}
	got := pt.Encode(t, ctx, input.Encode, nil)
	want := []byte{
		0x64, 0x00, 0x00, 0x00, // updateTime 100 (Encode4)  /*0x91be29*/
		0x02, 0x00, 0x00, 0x00, // count 2 (Encode4)         /*0x91be3b*/
		0x40, 0x00, 0x00, 0x00, // flag 0x40 (Encode4)       /*0x91be53*/
		0x01, 0x00, 0x00, 0x00, // value 1 (Encode4)         /*0x91be61*/
		0x80, 0x00, 0x00, 0x00, // flag 0x80 (Encode4)       /*0x91be53*/
		0x02, 0x00, 0x00, 0x00, // value 2 (Encode4)         /*0x91be61*/
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v72 AutoDistributeAp wire:\n got %x\nwant %x", got, want)
	}
}

// DistributeSp v72 byte-fixture — DISTRIBUTE_SP, op 89 (0x59).
//
// Client send — CWvsContext::SendSkillUpRequest sub_91BEAD @0x91BEAD:
//   COutPacket(89)                                                  /*0x91bed2*/
//   Encode4(update_time)                                            /*0x91bee4*/
//   Encode4(skillId)  // a2                                         /*0x91beef*/
//
// Body = updateTime(4) + skillId(4). == v79.
//
// packet-audit:verify packet=character/serverbound/DistributeSp version=gms_v72 ida=0x91bead
func TestDistributeSpByteOutputV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	input := DistributeSp{updateTime: 100, skillId: 1000000}
	got := pt.Encode(t, ctx, input.Encode, nil)
	want := []byte{
		0x64, 0x00, 0x00, 0x00, // updateTime 100 (Encode4)          /*0x91bee4*/
		0x40, 0x42, 0x0F, 0x00, // skillId 1000000=0xF4240 (Encode4) /*0x91beef*/
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v72 DistributeSp wire:\n got %x\nwant %x", got, want)
	}
}

// InfoRequest v72 byte-fixture — CHAR_INFO_REQUEST, op 96 (0x60).
//
// Client send — CWvsContext::SendCharacterInfoRequest sub_91C174 @0x91C174:
//   COutPacket(96)                                                  /*0x91c1d1*/
//   Encode4(update_time)                                            /*0x91c1e8*/
//   Encode4(characterId)  // v6                                     /*0x91c1f1*/
//   Encode1(petInfo)  // a4                                         /*0x91c1fc*/
//
// Body = updateTime(4) + characterId(4) + petInfo(1). == v79.
//
// packet-audit:verify packet=character/serverbound/InfoRequest version=gms_v72 ida=0x91c174
func TestInfoRequestByteOutputV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	input := InfoRequest{updateTime: 100, characterId: 12345, petInfo: true}
	got := pt.Encode(t, ctx, input.Encode, nil)
	want := []byte{
		0x64, 0x00, 0x00, 0x00, // updateTime 100 (Encode4)        /*0x91c1e8*/
		0x39, 0x30, 0x00, 0x00, // characterId 12345=0x3039 (Enc4) /*0x91c1f1*/
		0x01, // petInfo true (Encode1)                            /*0x91c1fc*/
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v72 InfoRequest wire:\n got %x\nwant %x", got, want)
	}
}

// HealOverTime v72 byte-fixture — HEAL_OVER_TIME, op 88 (0x58).
//
// Client send — CWvsContext::SendStatChangeRequest @0x9179C6:
//   COutPacket(88)                                                  /*0x9179d8*/
//   Encode4(0x1400)  // val = constant HP|MP mask                   /*0x9179e9*/
//   Encode2(hp)                                                     /*0x9179f4*/
//   Encode2(mp)                                                     /*0x9179ff*/
//   Encode1(option)                                                 /*0x917a0a*/
//
// There is NO get_update_time call → no leading updateTime dword (legacy GMS <83,
// heal_over_time.go). Body = val(4) + hp(2) + mp(2) + option(1) = 9 bytes. == v79.
//
// packet-audit:verify packet=character/serverbound/HealOverTime version=gms_v72 ida=0x9179c6
func TestHealOverTimeByteOutputV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	input := HealOverTime{updateTime: 100, val: 0x1400, hp: 50, mp: 30, unknown: 1}
	got := pt.Encode(t, ctx, input.Encode, nil)
	want := []byte{
		0x00, 0x14, 0x00, 0x00, // val 0x1400 (Encode4)   /*0x9179e9*/
		0x32, 0x00, // hp 50 (Encode2)                    /*0x9179f4*/
		0x1E, 0x00, // mp 30 (Encode2)                    /*0x9179ff*/
		0x01, // option (Encode1)                          /*0x917a0a*/
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v72 HealOverTime wire:\n got %x\nwant %x", got, want)
	}
}
