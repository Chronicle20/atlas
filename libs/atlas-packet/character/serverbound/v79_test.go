package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// v79 character-management serverbound byte fixtures (GMS_v79_1_DEVM.exe, port 13340).

// CheckName v79 byte-fixture.
//
// Client send — CLogin::SendCheckDuplicateIDPacket sub_5CD111 @0x5cd111:
//
//	COutPacket(21) then EncodeStr(name) /*0x5cd15a,0x5cd177*/. Single name string.
//
// packet-audit:verify packet=character/serverbound/CheckName version=gms_v79 ida=0x5cd111
func TestCheckNameByteOutputV79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	got := CheckName{name: "TestChar"}.Encode(nil, ctx)(nil)
	want := []byte{
		0x08, 0x00, // EncodeStr length = 8       /*0x5cd177*/
		0x54, 0x65, 0x73, 0x74, 0x43, 0x68, 0x61, 0x72, // "TestChar"
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v79 CheckName wire: got %x want %x", got, want)
	}
}

// CreateCharacter v79 byte-fixture.
//
// Client send — CLogin::SendNewCharPacket sub_5CCFA4 @0x5ccfa4:
//
//	COutPacket(22)                                                  /*0x5cd010*/
//	EncodeStr(name)                                                 /*0x5cd029*/
//	Encode4(jobIndex)                                               /*0x5cd037*/
//	loop 8x Encode4 (face, hair, hairColor, skinColor, top, bottom, shoes, weapon) /*0x5cd03f..0x5cd051*/
//	Encode1(gender)                                                 /*0x5cd05f*/
//
// v79 gates (GMS 79): >=73 writes jobIndex; <87 writes no subJob short; >28
// writes gender byte. The 9 ints (jobIndex + 8-int loop) and gender match the
// codec's 9 WriteInt + 1 WriteByte exactly.
//
// packet-audit:verify packet=character/serverbound/CreateCharacter version=gms_v79 ida=0x5ccfa4
func TestCreateCharacterByteOutputV79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	input := CreateCharacter{
		name:             "TestChar",
		jobIndex:         1,
		subJobIndex:      0,
		face:             20000, // 0x4E20
		hair:             30000, // 0x7530
		hairColor:        0,
		skinColor:        0,
		topTemplateId:    1040002, // 0x0FE0C2
		bottomTemplateId: 1060002, // 0x102CA2
		shoesTemplateId:  1072001, // 0x105D01
		weaponTemplateId: 1302000, // 0x13DD30
		gender:           0,
	}
	got := input.Encode(nil, ctx)(nil)
	want := []byte{
		0x08, 0x00, // name length = 8            /*0x5cd029*/
		0x54, 0x65, 0x73, 0x74, 0x43, 0x68, 0x61, 0x72, // "TestChar"
		0x01, 0x00, 0x00, 0x00, // jobIndex = 1   /*0x5cd037*/
		0x20, 0x4e, 0x00, 0x00, // face = 20000   /*0x5cd048 loop*/
		0x30, 0x75, 0x00, 0x00, // hair = 30000
		0x00, 0x00, 0x00, 0x00, // hairColor = 0
		0x00, 0x00, 0x00, 0x00, // skinColor = 0
		0x82, 0xde, 0x0f, 0x00, // top = 1040002
		0xa2, 0x2c, 0x10, 0x00, // bottom = 1060002
		0x81, 0x5b, 0x10, 0x00, // shoes = 1072001
		0xf0, 0xdd, 0x13, 0x00, // weapon = 1302000
		0x00, // gender                           /*0x5cd05f*/
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v79 CreateCharacter wire:\n got %x\nwant %x", got, want)
	}
}

// DeleteCharacter v79 byte-fixture.
//
// Client send — CLogin::SendDeleteCharPacket sub_5CCE4B @0x5cce4b:
//
//	COutPacket(23)                                                  /*0x5ccf1a*/
//	Encode4(dob)   // date-of-birth security value                  /*0x5ccf28*/
//	Encode4(characterId)                                            /*0x5ccf45*/
//
// v79 (<=82) uses the DOB path (no PIC string). Matches DeleteCharacter.Encode
// GMS<=82 branch ([int dob][int characterId]).
//
// packet-audit:verify packet=character/serverbound/DeleteCharacter version=gms_v79 ida=0x5cce4b
func TestDeleteCharacterByteOutputV79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	input := DeleteCharacter{dob: 19900101, characterId: 12345}
	got := input.Encode(nil, ctx)(nil)
	want := []byte{
		0xc5, 0xa6, 0x2f, 0x01, // dob = 19900101 (Decode4) /*0x5ccf28*/
		0x39, 0x30, 0x00, 0x00, // characterId = 12345      /*0x5ccf45*/
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v79 DeleteCharacter wire: got %x want %x", got, want)
	}
}

// ChairFixed v79 byte-fixture — CANCEL_CHAIR serverbound, op 40.
//
// Client send — CUserLocal::HandleXKeyDown @0x8a6e56 (the sit-on/get-up-from
// map-seat request): COutPacket(40) @0x8a6f4b then Encode2(v3) @0x8a6f58, where
// v3 is the resolved seat index (or 0xFFFF to get up, matching
// CWvsContext::SendGetUpFromChairRequest @0x95b4eb: COutPacket(40) +
// Encode2(0xFFFF)). Single int16, body only (opcode framing is out of scope).
// Matches ChairFixed.Encode ([int16 chairId]) exactly.
//
// packet-audit:verify packet=character/serverbound/ChairFixed version=gms_v79 ida=0x8a6e56
func TestChairFixedByteOutputV79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	got := ChairFixed{chairId: 42}.Encode(nil, ctx)(nil)
	want := []byte{
		0x2a, 0x00, // chairId 42 (Encode2 @0x8a6f58) /*0x8a6f58*/
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v79 ChairFixed wire: got %x want %x", got, want)
	}
}

// ---------------------------------------------------------------------------
// Stage E batch 18 — character part G (keymap / friendly-damage serverbound).
// ---------------------------------------------------------------------------

// KeyMapChange v79 byte-fixture — CHANGE_KEYMAP, op 132.
//
// Client send — CFuncKeyMappedMan::SaveFuncKeyMap @0x569fe4:
//
//	COutPacket(132)                                                  /*0x569ffe*/
//	Encode4(0)                             // mode (always 0)         /*0x56a00c*/
//	Encode4(count)                         // # changed keys          /*0x56a068*/
//	per changed key:
//	    Encode4(keyIdx)                    // key index               /*0x56a07c*/
//	    FUNCKEY_MAPPED::Encode             // EncodeBuffer(5) = nType[1]+nID[4] /*0x56a092*/
//
// Byte-identical to v83 (mode int32 + count int32 + per-entry [keyId int32 +
// theType int8 + action int32] = 9 bytes/entry). No version gate in the codec.
//
// packet-audit:verify packet=character/serverbound/KeyMapChange version=gms_v79 ida=0x569fe4
func TestKeyMapChangeByteOutputV79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	input := KeyMapChange{
		mode: 0,
		entries: []KeyMapEntry{
			{KeyId: 2, TheType: 4, Action: 10},
			{KeyId: 16, TheType: 4, Action: 8},
		},
	}
	got := input.Encode(nil, ctx)(nil)
	want := []byte{
		0x00, 0x00, 0x00, 0x00, // mode = 0 (Encode4)            /*0x56a00c*/
		0x02, 0x00, 0x00, 0x00, // count = 2 (Encode4)           /*0x56a068*/
		0x02, 0x00, 0x00, 0x00, // keyIdx 2 (Encode4)            /*0x56a07c*/
		0x04,                   // theType 4 (FUNCKEY byte 0)     /*0x56a092*/
		0x0a, 0x00, 0x00, 0x00, // action 10 (FUNCKEY bytes 1-4)
		0x10, 0x00, 0x00, 0x00, // keyIdx 16 (Encode4)           /*0x56a07c*/
		0x04,                   // theType 4
		0x08, 0x00, 0x00, 0x00, // action 8
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v79 KeyMapChange wire:\n got %x\nwant %x", got, want)
	}
}

// MonsterDamageFriendly v79 byte-fixture — MOB_DAMAGE_MOB_FRIENDLY, op 184
// (and FIELD_DAMAGE_MOB op 183; both map to this struct in the matrix).
//
// Client send — CMob::Update (v79 sub_6361DF @0x6361df), friendly-damage send
// site @0x636b8f:
//
//	COutPacket(184)                                                 /*0x636b8f*/
//	Encode4(SecureFuse(this.m_dwMobID))    // attackerId (friendly mob) /*0x636bb3*/
//	Encode4(CWvsContext.dwCharacterID)     // observerId (controller)   /*0x636bc6*/
//	Encode4(SecureFuse(attacker.m_dwMobID))// attackedId (hostile mob)  /*0x636be3*/
//
// Three Encode4, no version gate — byte-identical to the v83 golden fixture.
//
// packet-audit:verify packet=character/serverbound/CharacterMonsterDamageFriendly version=gms_v79 ida=0x6361df
func TestMonsterDamageFriendlyByteOutputV79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	input := MonsterDamageFriendly{attackerId: 0x11223344, observerId: 0x0010F447, attackedId: 0xAABBCCDD}
	got := input.Encode(nil, ctx)(nil)
	want := []byte{
		0x44, 0x33, 0x22, 0x11, // attackerId (Encode4 this.m_dwMobID)     /*0x636bb3*/
		0x47, 0xF4, 0x10, 0x00, // observerId (Encode4 dwCharacterID)      /*0x636bc6*/
		0xDD, 0xCC, 0xBB, 0xAA, // attackedId (Encode4 attacker.m_dwMobID) /*0x636be3*/
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v79 MonsterDamageFriendly wire:\n got % x\nwant % x", got, want)
	}
}
