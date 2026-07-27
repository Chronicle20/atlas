package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// v61 character serverbound byte fixtures (GMS_v61.1_U_DEVM.exe, port 13338).
//
// task-113: the CWvsContext ability/social serverbound region was scrambled by
// the Stage-B harvest; opcodes re-derived from each COutPacket(N) send-site body
// + caller and cross-checked against the verified gms_v72 twins. See
// docs/packets/registry/gms_v61.yaml. AUTO_DISTRIBUTE_AP is v61-ABSENT (no
// array/loop Encode4 sender exists in the whole IDB), so it carries no v61
// fixture — the matrix leaves the cell blank (verified absent).

// CheckName v61 byte-fixture — CHECK_CHAR_NAME, op 21 (0x15).
//
// Client send — sub_565537 @0x565537 (v72 CLogin::SendCheckDuplicateIDPacket
// twin): COutPacket(21) @0x56558b then EncodeStr(name) @0x5655a8. Single name
// string, no version divergence. == v72.
//
// packet-audit:verify packet=character/serverbound/CheckName version=gms_v61 ida=0x565537
func TestCheckNameByteOutputV61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	got := CheckName{name: "TestChar"}.Encode(nil, ctx)(nil)
	want := []byte{
		0x08, 0x00, // EncodeStr length = 8       /*0x5655a8*/
		0x54, 0x65, 0x73, 0x74, 0x43, 0x68, 0x61, 0x72, // "TestChar"
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v61 CheckName wire: got %x want %x", got, want)
	}
}

// CreateCharacter v61 byte-fixture — CREATE_CHAR, op 22 (0x16).
//
// Client send — sub_5653E9 @0x5653e9:
//
//	COutPacket(22)                                                   /*0x56542f*/
//	EncodeStr(name)                                                  /*0x56544c*/
//	loop 8x Encode4 (face,hair,hairColor,skinColor,top,bottom,shoes,weapon) /*0x565463*/
//	Encode1(gender)                                                  /*0x565482*/
//	do 4x Encode1(a3[i*4])  // str,dex,int,luk low bytes             /*0x565495*/
//
// LEGACY DIVERGENCE vs v72: v61 (<=61) trails the four manually-rolled base
// stats (str/dex/int/luk) after gender; v72 SendNewCharPacket sub_5B219A ends at
// gender. create.go gates the four stat bytes on GMS<=61 (task-113). No jobIndex
// (GMS<73) and no subJobIndex (GMS<87).
//
// packet-audit:verify packet=character/serverbound/CreateCharacter version=gms_v61 ida=0x5653e9
func TestCreateCharacterByteOutputV61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	input := CreateCharacter{
		name:             "TestChar",
		jobIndex:         1,     // NOT written for v61 (<73)
		subJobIndex:      0,     // NOT written for v61 (<87)
		face:             20000, // 0x4E20
		hair:             30000, // 0x7530
		hairColor:        0,
		skinColor:        0,
		topTemplateId:    1040002, // 0x0FDE82
		bottomTemplateId: 1060002, // 0x102CA2
		shoesTemplateId:  1072001, // 0x105D81
		weaponTemplateId: 1302000, // 0x13DDF0
		gender:           0,
		strength:         12,
		dexterity:        5,
		intelligence:     4,
		luck:             4,
	}
	got := input.Encode(nil, ctx)(nil)
	want := []byte{
		0x08, 0x00, // name length = 8            /*0x56544c*/
		0x54, 0x65, 0x73, 0x74, 0x43, 0x68, 0x61, 0x72, // "TestChar"
		0x20, 0x4e, 0x00, 0x00, // face = 20000   /*0x565463 loop*/
		0x30, 0x75, 0x00, 0x00, // hair = 30000
		0x00, 0x00, 0x00, 0x00, // hairColor = 0
		0x00, 0x00, 0x00, 0x00, // skinColor = 0
		0x82, 0xde, 0x0f, 0x00, // top = 1040002
		0xa2, 0x2c, 0x10, 0x00, // bottom = 1060002
		0x81, 0x5b, 0x10, 0x00, // shoes = 1072001
		0xf0, 0xdd, 0x13, 0x00, // weapon = 1302000
		0x00, // gender                           /*0x565482*/
		0x0c, // strength = 12                    /*0x565495 loop*/
		0x05, // dexterity = 5
		0x04, // intelligence = 4
		0x04, // luck = 4
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v61 CreateCharacter wire:\n got %x\nwant %x", got, want)
	}
}

// DeleteCharacter v61 byte-fixture — DELETE_CHAR, op 23 (0x17).
//
// Client send — sub_5652E3 @0x5652e3:
//
//	COutPacket(23)                                                   /*0x56535f*/
//	Encode4(dob)   // date-of-birth security value                  /*0x56536d*/
//	Encode4(characterId)                                            /*0x56538a*/
//
// Body = [int dob][int characterId] == v72 body (GMS<83 → dob, not PIC). v61 uses
// op 23 (the v83-style DELETE_CHAR), one below v72's op 24.
//
// packet-audit:verify packet=character/serverbound/DeleteCharacter version=gms_v61 ida=0x5652e3
func TestDeleteCharacterByteOutputV61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	input := DeleteCharacter{dob: 19900101, characterId: 12345}
	got := input.Encode(nil, ctx)(nil)
	want := []byte{
		0xc5, 0xa6, 0x2f, 0x01, // dob = 19900101 (Encode4)   /*0x56536d*/
		0x39, 0x30, 0x00, 0x00, // characterId = 12345         /*0x56538a*/
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v61 DeleteCharacter wire: got %x want %x", got, want)
	}
}

// DistributeAp v61 byte-fixture — DISTRIBUTE_AP, op 80 (0x50).
//
// Client send — sub_8457EE @0x8457ee (v72 SendAbilityUpRequest(DWORD) sub_91BBAD
// twin; ForcedStat guard + remaining-AP check): COutPacket(80) @0x845889,
// Encode4(update_time) @0x84589a, Encode4(dwFlag) @0x8458a5. Caller sub_72DFBB =
// stat +buttons passing STR 0x40/DEX 0x80/INT 0x100/LUK 0x200/HP 0x800/MP 0x2000.
// Body = updateTime(4) + dwFlag(4).
//
// packet-audit:verify packet=character/serverbound/DistributeAp version=gms_v61 ida=0x8457ee
func TestDistributeApByteOutputV61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	input := DistributeAp{updateTime: 100, dwFlag: 0x40}
	got := input.Encode(nil, ctx)(nil)
	want := []byte{
		0x64, 0x00, 0x00, 0x00, // updateTime 100 (Encode4)   /*0x84589a*/
		0x40, 0x00, 0x00, 0x00, // dwFlag 0x40=STR (Encode4)  /*0x8458a5*/
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v61 DistributeAp wire:\n got %x\nwant %x", got, want)
	}
}

// DistributeSp v61 byte-fixture — DISTRIBUTE_SP, op 82 (0x52).
//
// Client send — sub_8458EB @0x8458eb (v72 SendSkillUpRequest sub_91BEAD twin;
// caller sub_7201E6 = the skill-window "+" validator): COutPacket(82) @0x845912,
// Encode4(update_time) @0x845924, Encode4(skillId) @0x84592f. Body =
// updateTime(4) + skillId(4).
//
// packet-audit:verify packet=character/serverbound/DistributeSp version=gms_v61 ida=0x8458eb
func TestDistributeSpByteOutputV61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	input := DistributeSp{updateTime: 100, skillId: 1000000}
	got := input.Encode(nil, ctx)(nil)
	want := []byte{
		0x64, 0x00, 0x00, 0x00, // updateTime 100 (Encode4)          /*0x845924*/
		0x40, 0x42, 0x0F, 0x00, // skillId 1000000=0xF4240 (Encode4) /*0x84592f*/
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v61 DistributeSp wire:\n got %x\nwant %x", got, want)
	}
}

// InfoRequest v61 byte-fixture — CHAR_INFO_REQUEST, op 89 (0x59).
//
// Client send — sub_845B68 @0x845b68 (v72 SendCharacterInfoRequest sub_91C174
// twin; CUserPool::GetRemoteUser): COutPacket(89) @0x845bc5,
// Encode4(update_time) @0x845bdc, Encode4(characterId) @0x845be5,
// Encode1(petInfo) @0x845bf0. Body = updateTime(4) + characterId(4) + petInfo(1).
//
// packet-audit:verify packet=character/serverbound/InfoRequest version=gms_v61 ida=0x845b68
func TestInfoRequestByteOutputV61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	input := InfoRequest{updateTime: 100, characterId: 12345, petInfo: true}
	got := input.Encode(nil, ctx)(nil)
	want := []byte{
		0x64, 0x00, 0x00, 0x00, // updateTime 100 (Encode4)        /*0x845bdc*/
		0x39, 0x30, 0x00, 0x00, // characterId 12345=0x3039 (Enc4) /*0x845be5*/
		0x01, // petInfo true (Encode1)                            /*0x845bf0*/
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v61 InfoRequest wire:\n got %x\nwant %x", got, want)
	}
}

// HealOverTime v61 byte-fixture — HEAL_OVER_TIME, op 81 (0x51).
//
// Client send — CWvsContext::SendStatChangeRequest @0x8421f0:
//
//	COutPacket(81)                                                   /*0x842204*/
//	Encode4(0x1400)  // constant HP|MP mask                         /*0x842215*/
//	Encode2(hp)                                                     /*0x842220*/
//	Encode2(mp)                                                     /*0x84222b*/
//	Encode1(option)                                                 /*0x842236*/
//
// No get_update_time → no leading updateTime dword (legacy GMS<83,
// heal_over_time.go). Body = val(4) + hp(2) + mp(2) + option(1) = 9 bytes. == v72.
//
// packet-audit:verify packet=character/serverbound/HealOverTime version=gms_v61 ida=0x8421f0
func TestHealOverTimeByteOutputV61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	input := HealOverTime{updateTime: 100, val: 0x1400, hp: 50, mp: 30, unknown: 1}
	got := input.Encode(nil, ctx)(nil)
	want := []byte{
		0x00, 0x14, 0x00, 0x00, // val 0x1400 (Encode4)   /*0x842215*/
		0x32, 0x00, // hp 50 (Encode2)                    /*0x842220*/
		0x1E, 0x00, // mp 30 (Encode2)                    /*0x84222b*/
		0x01, // option (Encode1)                          /*0x842236*/
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v61 HealOverTime wire:\n got %x\nwant %x", got, want)
	}
}

// ChairFixed v61 byte-fixture — CANCEL_CHAIR serverbound, op 39.
//
// Client send — CWvsContext::SendGetUpFromChairRequest @0x8374FE:
//
//	COutPacket(39)                                                  /*0x837523*/
//	Encode2(0xFFFF)   // seat index; 0xFFFF (-1) = get-up-from-chair /*0x837534*/
//
// Single int16 body == v72 ChairFixed.Encode ([int16 chairId]); the get-up path
// always sends 0xFFFF. v61 op 39 (v72 CANCEL_CHAIR=41, Δ-2).
//
// packet-audit:verify packet=character/serverbound/ChairFixed version=gms_v61 ida=0x8374fe
func TestChairFixedByteOutputV61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	got := ChairFixed{chairId: -1}.Encode(nil, ctx)(nil)
	want := []byte{0xff, 0xff} // chairId 0xFFFF (Encode2) /*0x837534*/
	if !bytes.Equal(got, want) {
		t.Errorf("v61 ChairFixed wire: got %x want %x", got, want)
	}
}

// KeyMapChange v61 byte-fixture — CHANGE_KEYMAP, op 123 (0x7B).
//
// Client send — CFuncKeyMappedMan::SaveFuncKeyMap @0x51AC0D:
//
//	COutPacket(123)                                                 /*0x51ac27*/
//	Encode4(0)                             // mode (always 0)        /*0x51ac33*/
//	Encode4(count)                         // # changed keys         /*0x51ac80*/
//	per changed key:
//	    Encode4(keyIdx)                    // key index              /*0x51ac94*/
//	    FUNCKEY_MAPPED::Encode             // nType[1]+nID[4]        /*0x51acaa*/
//
// mode int32 + count int32 + per-entry [keyId int32 + theType int8 + action int32].
// No version gate; byte-identical to v72.
//
// packet-audit:verify packet=character/serverbound/KeyMapChange version=gms_v61 ida=0x51ac0d
func TestKeyMapChangeByteOutputV61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	input := KeyMapChange{
		mode: 0,
		entries: []KeyMapEntry{
			{KeyId: 2, TheType: 4, Action: 10},
			{KeyId: 16, TheType: 4, Action: 8},
		},
	}
	got := input.Encode(nil, ctx)(nil)
	want := []byte{
		0x00, 0x00, 0x00, 0x00, // mode = 0 (Encode4)      /*0x51ac33*/
		0x02, 0x00, 0x00, 0x00, // count = 2 (Encode4)     /*0x51ac80*/
		0x02, 0x00, 0x00, 0x00, // keyIdx 2 (Encode4)      /*0x51ac94*/
		0x04,                   // theType 4               /*0x51acaa*/
		0x0a, 0x00, 0x00, 0x00, // action 10
		0x10, 0x00, 0x00, 0x00, // keyIdx 16 (Encode4)
		0x04,                   // theType 4
		0x08, 0x00, 0x00, 0x00, // action 8
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v61 KeyMapChange wire:\n got %x\nwant %x", got, want)
	}
}

// ExpressionRequest v61 byte-fixture — FACE_EXPRESSION serverbound, op 48.
//
// Client send — CWvsContext::SendEmotionChange @0x845E8F:
//
//	COutPacket(48)                                                  /*0x845f27*/
//	Encode4(emotion)   // SecureFuse(avatar emotion); validated <= 0x17 /*0x845f4b*/
//
// v61 (GMS < 87) sends NO Encode4(duration) and NO Encode1(byItemOption) — both
// are GMS>87 additions. expression.go gates them on GMS>87. Body = emote(4).
//
// packet-audit:verify packet=character/serverbound/ExpressionRequest version=gms_v61 ida=0x845e8f
func TestExpressionRequestByteOutputV61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	got := ExpressionRequest{emote: 5}.Encode(nil, ctx)(nil)
	want := []byte{0x05, 0x00, 0x00, 0x00} // emote 5 (Encode4) /*0x845f4b*/
	if !bytes.Equal(got, want) {
		t.Errorf("v61 ExpressionRequest wire: got %x want %x", got, want)
	}
}

// DropMeso v61 byte-fixture — MESO_DROP, op 86 (0x56).
//
// Client send — CWvsContext::SendDropMoneyRequest (sub_8459DD) @0x8459DD:
//
//	COutPacket(86)                                                  /*0x845a04*/
//	Encode4(updateTime)                                             /*0x845a16*/
//	Encode4(amount)                                                 /*0x845a21*/
//
// Body = updateTime(4) + amount(4) == v72 MESO_DROP. task-113 scramble fix: real
// v61 MESO_DROP is op 86 (was mislabeled DISTRIBUTE_AP). Caller = drop-meso spinner.
//
// packet-audit:verify packet=character/serverbound/DropMeso version=gms_v61 ida=0x8459dd
func TestDropMesoByteOutputV61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	got := DropMeso{updateTime: 100, amount: 5000}.Encode(nil, ctx)(nil)
	want := []byte{
		0x64, 0x00, 0x00, 0x00, // updateTime 100 (Encode4)   /*0x845a16*/
		0x88, 0x13, 0x00, 0x00, // amount 5000 (Encode4)       /*0x845a21*/
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v61 DropMeso wire:\n got %x\nwant %x", got, want)
	}
}

// ChalkboardClose v61 byte-fixture — CLOSE_CHALKBOARD, op 47 (0x2F).
//
// Client send — sub_7A3981 @0x7A3981 (CUserLocal close-chalkboard on cursor):
//
//	COutPacket(47)                                                  /*0x7a39bc*/
//	SendPacket                             // NO body               /*0x7a39cf*/
//
// Empty body == v72. v61 op 47 (v72 CLOSE_CHALKBOARD=49, Δ-2).
//
// packet-audit:verify packet=character/serverbound/ChalkboardClose version=gms_v61 ida=0x7a3981
func TestChalkboardCloseByteOutputV61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	got := ChalkboardClose{}.Encode(nil, ctx)(nil)
	if len(got) != 0 {
		t.Errorf("v61 ChalkboardClose wire: got %x want empty", got)
	}
}

// MOB_DAMAGE_MOB_FRIENDLY (op159) — CMob::Update mob-vs-mob-friendly send site
// sub_5C71B7 @0x5c7aa3: COutPacket(159); Encode4(SecureFuse(this.m_dwMobID))
// @0x5c78d9→@0x5c7ac7 (attacker); Encode4(g_pWvsContext dwCharacterID +0x2088)
// @0x5c7ada (observer/local char); Encode4(SecureFuse(v48.m_dwMobID)) @0x5c7af7
// (attacked); SendPacket. Three Encode4, no version gate — byte-identical to the
// verified v72 anchor. v61 op159 = v72 op182 − 23.
//
// packet-audit:verify packet=character/serverbound/CharacterMonsterDamageFriendly version=gms_v61 ida=0x5c71b7
func TestMonsterDamageFriendlyByteOutputV61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	input := MonsterDamageFriendly{attackerId: 0x11223344, observerId: 0x0010F447, attackedId: 0xAABBCCDD}
	got := input.Encode(nil, ctx)(nil)
	want := []byte{
		0x44, 0x33, 0x22, 0x11, // attackerId (Encode4 @0x5c7ac7)
		0x47, 0xF4, 0x10, 0x00, // observerId (Encode4 dwCharacterID @0x5c7ada)
		0xDD, 0xCC, 0xBB, 0xAA, // attackedId (Encode4 @0x5c7af7)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v61 MonsterDamageFriendly wire:\n got % x\nwant % x", got, want)
	}
}
