package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// v79 character-management clientbound byte fixtures (GMS_v79_1_DEVM.exe, port 13340).
//
// The GW_CharacterStat and AvatarLook wire blocks are byte-identical to the v83
// fixture (list_test.go) — v79 shares every relevant version gate: >28 (3 pet
// longs + gachaExp), <95 (int16 HP/MP), >12 (trailing int), >=87 false (no
// nSubJob). So the two blocks below are copied verbatim from the verified v83
// fixture; only the top-level framing differs per handler.
//
// GW_CharacterStat::Decode (v79 @0x4d6e21, list path a3=0): id=Decode4,
// name=DecodeBuffer(13), gender=Decode1, skin=Decode1, face=Decode4, hair=Decode4,
// petLockerSN=DecodeBuffer(24), level=Decode1, 10x Decode2 (job,str,dex,int,luk,
// hp,maxHp,mp,maxMp,ap), sp=Decode2, exp=Decode4, fame=Decode2, gachaExp=Decode4,
// mapId=Decode4, spawnPoint=Decode1, trailing Decode4.
// AvatarLook::Decode (v79 @0x4db6dd): gender,skin,face,!mega,hair, equip loop
// (0xFF term), masked loop (0xFF term), cashWeapon=Decode4, pets=DecodeBuffer(12).

// heroStatsV79 / heroAvatarV79 mirror the v83 list fixture model exactly.
func heroStatsV79() model.CharacterStatistics {
	return model.NewCharacterStatistics(
		0x01020304, "Hero", 0, 0, 0x4D2, 0x7B, [3]uint64{0, 0, 0},
		0x0A, 0x64, 4, 5, 6, 7, 0x64, 0x64, 0x32, 0x32, 3, false, 2, 0, 8, 0, 0x0BB8, 0,
	)
}

func heroAvatarV79() model.Avatar {
	return model.NewAvatar(0, 0, 0x4D2, false, 0x7B, nil, nil, nil)
}

// statBlockV79 = the GW_CharacterStat wire for heroStatsV79 (== v83 fixture bytes).
var statBlockV79 = []byte{
	0x04, 0x03, 0x02, 0x01, // id (Decode4)                        /*0x4d6e44*/
	0x48, 0x65, 0x72, 0x6f, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // name (DecodeBuffer 13) /*0x4d6e55*/
	0x00,                   // gender (Decode1)                    /*0x4d6e6c*/
	0x00,                   // skin (Decode1)                      /*0x4d6e81*/
	0xd2, 0x04, 0x00, 0x00, // face (Decode4)                      /*0x4d6e96*/
	0x7b, 0x00, 0x00, 0x00, // hair (Decode4)                      /*0x4d6eab*/
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // pet long 0 (DecodeBuffer 24) /*0x4d6eb6*/
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // pet long 1
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // pet long 2
	0x0a,                   // level (Decode1)                     /*0x4d6ec5*/
	0x64, 0x00,             // job (Decode2)                       /*0x4d6ed1*/
	0x04, 0x00,             // str (Decode2)                       /*0x4d6ee5*/
	0x05, 0x00,             // dex                                 /*0x4d6ef9*/
	0x06, 0x00,             // int                                 /*0x4d6f0d*/
	0x07, 0x00,             // luck                                /*0x4d6f21*/
	0x64, 0x00,             // hp                                  /*0x4d6f35*/
	0x64, 0x00,             // maxHp                               /*0x4d6f49*/
	0x32, 0x00,             // mp                                  /*0x4d6f5d*/
	0x32, 0x00,             // maxMp                               /*0x4d6f71*/
	0x03, 0x00,             // ap (Decode2)                        /*0x4d6f85*/
	0x02, 0x00,             // sp (Decode2)                        /*0x4d6f9f*/
	0x00, 0x00, 0x00, 0x00, // exp (Decode4)                       /*0x4d6fb9*/
	0x08, 0x00,             // fame (Decode2)                      /*0x4d6fd3*/
	0x00, 0x00, 0x00, 0x00, // gachaExp (Decode4)                  /*0x4d6fed*/
	0xb8, 0x0b, 0x00, 0x00, // mapId (Decode4)                     /*0x4d7007*/
	0x00,                   // spawnPoint (Decode1)                /*0x4d7028*/
	0x00, 0x00, 0x00, 0x00, // trailing int (Decode4)              /*0x4d702e*/
}

// avatarBlockV79 = the AvatarLook wire for heroAvatarV79 (== v83 fixture bytes).
var avatarBlockV79 = []byte{
	0x00,                   // gender (Decode1)                    /*0x4db6f0*/
	0x00,                   // skin (Decode1)                      /*0x4db6fd*/
	0xd2, 0x04, 0x00, 0x00, // face (Decode4)                      /*0x4db711*/
	0x01,                   // !mega (Decode1)                     /*0x4db72d*/
	0x7b, 0x00, 0x00, 0x00, // hair (Decode4)                      /*0x4db739*/
	0xff,                   // equip terminator                    /*0x4db742*/
	0xff,                   // masked terminator                   /*0x4db779*/
	0x00, 0x00, 0x00, 0x00, // cash weapon (Decode4)               /*0x4db7b5*/
	0x00, 0x00, 0x00, 0x00, // pet 0 (DecodeBuffer 12)             /*0x4db7c8*/
	0x00, 0x00, 0x00, 0x00, // pet 1
	0x00, 0x00, 0x00, 0x00, // pet 2
}

// rank16V79 = the 4 rank ints (rank=1, rankMove=2, jobRank=3, jobRankMove=4) that
// the v79 client reads via DecodeBuffer(16) when the rankEnabled byte is non-zero.
var rank16V79 = []byte{
	0x01, 0x00, 0x00, 0x00, // rank
	0x02, 0x00, 0x00, 0x00, // rankMove
	0x03, 0x00, 0x00, 0x00, // jobRank
	0x04, 0x00, 0x00, 0x00, // jobRankMove
}

// CharacterList v79 byte-fixture.
//
// Client read order — v79 char-list decoder sub_5CE522 @0x5CE522, world-select
// success path (status 0/12/23, LABEL_32 @0x5ce6b3):
//
//	status = Decode1                     // result/status byte             /*0x5ce57a*/
//	count  = Decode1                     // number of avatar entries        /*0x5ce6e8*/
//	for each entry (12 slots, count decoded):
//	    GW_CharacterStat::Decode         // @0x4d6e21                       /*0x5ce719*/
//	    AvatarLook::Decode               // @0x4db6dd                       /*0x5ce727*/
//	    family = Decode1 (stored)        // viewAll/family flag byte         /*0x5ce743*/
//	    rankEnabled = Decode1            // 0 => memset 16; else buffer      /*0x5ce745*/
//	        if rankEnabled: DecodeBuffer(16) = 4 rank ints                   /*0x5ce760*/
//	slots  = Decode4                     // m_nSlotCount — NO hasPic byte    /*0x5ce7ac*/
//
// The legacy divergence vs v83+: the v79 client reads slots directly after the
// entry loop with NO login-option (hasPic) byte. list.go skips hasPic for GMS<83.
//
// packet-audit:verify packet=character/clientbound/CharacterList version=gms_v79 ida=0x5ce522
func TestCharacterListByteOutputV79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	entry := model.NewCharacterListEntry(heroStatsV79(), heroAvatarV79(), false, false, 1, 2, 3, 4)
	got := NewCharacterList(0, []model.CharacterListEntry{entry}, false, 8).Encode(nil, ctx)(nil)

	var want []byte
	want = append(want, 0x00) // status               /*0x5ce57a*/
	want = append(want, 0x01) // count = 1             /*0x5ce6e8*/
	want = append(want, statBlockV79...)
	want = append(want, avatarBlockV79...)
	want = append(want, 0x00)          // family/viewAll flag  /*0x5ce743*/
	want = append(want, 0x01)          // rankEnabled = !gm    /*0x5ce745*/
	want = append(want, rank16V79...)  // rank ints            /*0x5ce760*/
	want = append(want, 0x08, 0x00, 0x00, 0x00) // slots — no hasPic /*0x5ce7ac*/

	if !bytes.Equal(got, want) {
		t.Errorf("v79 CharacterList wire:\n got %x\nwant %x", got, want)
	}
}

// CharacterViewAllCharacters v79 byte-fixture.
//
// Client read order — CLogin::OnViewAllCharResult @0x5cee77, mode 0 (per-world
// character batch, case 0u @0x5ceead):
//
//	mode   = Decode1                     // dispatcher mode (0 = char batch) /*0x5ceead*/
//	worldId= Decode1                     // v28                              /*0x5cf0cb*/
//	count  = Decode1                     // v13                              /*0x5cf0db*/
//	for each of count entries:
//	    GW_CharacterStat::Decode         // @0x4d6e21                        /*0x5cf11b*/
//	    AvatarLook::Decode               // @0x4db6dd                        /*0x5cf129*/
//	    // worldId stored locally (NOT read); NO family byte (viewAll)
//	    rankEnabled = Decode1            // 0 => memset 16; else buffer       /*0x5cf13d*/
//	        if rankEnabled: DecodeBuffer(16) = 4 rank ints                    /*0x5cf158*/
//	// no trailing PIC byte (GMS<=87)
//
// The atlas "code" field maps to the dispatcher mode byte (0). Entries carry
// viewAll=true so CharacterListEntry omits the family placeholder byte.
//
// packet-audit:verify packet=character/clientbound/CharacterViewAllCharacters version=gms_v79 ida=0x5cee77
func TestCharacterViewAllCharactersByteOutputV79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	// viewAll=true entry: CharacterListEntry.Encode skips the family placeholder.
	entry := model.NewCharacterListEntry(heroStatsV79(), heroAvatarV79(), true, false, 1, 2, 3, 4)
	got := NewCharacterViewAllCharacters(0, world.Id(1), []model.CharacterListEntry{entry}).Encode(nil, ctx)(nil)

	var want []byte
	want = append(want, 0x00) // mode/code (case 0)   /*0x5ceead*/
	want = append(want, 0x01) // worldId              /*0x5cf0cb*/
	want = append(want, 0x01) // count = 1            /*0x5cf0db*/
	want = append(want, statBlockV79...)
	want = append(want, avatarBlockV79...)
	want = append(want, 0x01)         // rankEnabled = !gm (no family byte) /*0x5cf13d*/
	want = append(want, rank16V79...) // rank ints                         /*0x5cf158*/

	if !bytes.Equal(got, want) {
		t.Errorf("v79 CharacterViewAllCharacters wire:\n got %x\nwant %x", got, want)
	}
}

// AddCharacterEntry v79 byte-fixture.
//
// Client read order — v79 create-character result @0x5ceb55 (dispatcher case 14,
// mislabeled OnDeleteCharacterResult in the IDB; behaviorally the add handler):
//
//	code = Decode1                       // result code                     /*0x5ceb70*/
//	// on success (code 0, slot found):
//	GW_CharacterStat::Decode             // @0x4d6e21 into empty slot        /*0x5cec0e*/
//	AvatarLook::Decode                   // @0x4db6dd                        /*0x5cec1c*/
//	// family byte + 16-byte rank buffer are ZEROED locally, NOT read        /*0x5cec2a,0x5cec3e*/
//
// The legacy divergence vs v83+: the v79 add result has NO entry trailer
// (family/rank). add_entry.go writes only stat+avatar for GMS v29..v82.
//
// CORRECTED (task-113 stage E): the v79 IDB SYMBOL names for the three
// char-management handlers are rotated one step off their actual bodies. By
// handler-body behavior v79 is IDENTICAL to v83 — op13=CHAR_NAME_RESPONSE
// [string][byte] @0x5ce875, op14=ADD_NEW_CHAR_ENTRY [byte][stat][avatar]
// @0x5ceb55, op15=DELETE_CHAR_RESPONSE [int][byte] @0x5ce90a. registry/gms_v79.yaml
// and template_gms_79_1.json now carry the behavior-correct mapping (Add=0x0E,
// Delete=0x0F, Name=0x0D). The prior permuted mapping trusted the rotated symbols.
//
// packet-audit:verify packet=character/clientbound/AddCharacterEntry version=gms_v79 ida=0x5ceb55
func TestAddCharacterEntryByteOutputV79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	entry := model.NewCharacterListEntry(heroStatsV79(), heroAvatarV79(), false, false, 1, 2, 3, 4)
	got := NewAddCharacterEntry(0, entry).Encode(nil, ctx)(nil)

	var want []byte
	want = append(want, 0x00) // code                 /*0x5ceb70*/
	want = append(want, statBlockV79...)
	want = append(want, avatarBlockV79...)
	// no entry trailer

	if !bytes.Equal(got, want) {
		t.Errorf("v79 AddCharacterEntry wire:\n got %x\nwant %x", got, want)
	}
}

// DeleteCharacterResponse v79 byte-fixture.
//
// Client read order — v79 delete-character result @0x5ce90a (dispatcher case 15,
// mislabeled OnCheckDuplicatedIDResult in the IDB; behaviorally the delete
// handler — it removes the character with the decoded id from the list):
//
//	characterId = Decode4                // id of the char to remove        /*0x5ce92d*/
//	code        = Decode1                // result code                     /*0x5ce930*/
//
// Matches DeleteCharacterResponse.Encode exactly ([int charId][byte code]).
//
// CORRECTED (task-113 stage E): rotated-symbol fix — DELETE_CHAR_RESPONSE is
// v79 op 15 (0x0F), same as v83, per the case-15 handler body @0x5ce90a.
//
// packet-audit:verify packet=character/clientbound/DeleteCharacterResponse version=gms_v79 ida=0x5ce90a
func TestDeleteCharacterResponseByteOutputV79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	got := NewDeleteCharacterResponse(12345, 0).Encode(nil, ctx)(nil)
	want := []byte{
		0x39, 0x30, 0x00, 0x00, // characterId = 12345 (Decode4) /*0x5ce92d*/
		0x00, // code (Decode1)                                  /*0x5ce930*/
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v79 DeleteCharacterResponse wire: got %x want %x", got, want)
	}
}

// CharacterNameResponse v79 byte-fixture.
//
// Client read order — v79 name-check result @0x5ce875 (dispatcher case 13,
// mislabeled OnCreateNewCharacterResult in the IDB; behaviorally the duplicate-id
// check result — it reads the echoed name string then the availability code):
//
//	name = DecodeStr                     // echoed character name           /*0x5ce894*/
//	code = Decode1                       // availability/result code        /*0x5ce89f*/
//
// Matches CharacterNameResponse.Encode exactly ([string name][byte code]).
//
// CORRECTED (task-113 stage E): rotated-symbol fix — CHAR_NAME_RESPONSE is
// v79 op 13 (0x0D), same as v83, per the case-13 handler body @0x5ce875.
//
// packet-audit:verify packet=character/clientbound/CharacterNameResponse version=gms_v79 ida=0x5ce875
func TestCharacterNameResponseByteOutputV79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	got := NewCharacterNameResponse("TestChar", 0).Encode(nil, ctx)(nil)
	want := []byte{
		0x08, 0x00, // name length = 8 (DecodeStr)               /*0x5ce894*/
		0x54, 0x65, 0x73, 0x74, 0x43, 0x68, 0x61, 0x72, // "TestChar"
		0x00, // code (Decode1)                                  /*0x5ce89f*/
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v79 CharacterNameResponse wire: got %x want %x", got, want)
	}
}

// The remaining three VIEW_ALL_CHAR sub-modes share CLogin::OnViewAllCharResult
// @0x5cee77 with CharacterViewAllCharacters; the leading Decode1 (mode/code
// @0x5ceead) selects the branch. Like every other version (v83 @0x5facca etc.),
// all sub-writers pin against the same base function address.

// CharacterViewAllCount v79 byte-fixture — mode 1 (case 1u @0x5ceead):
//	mode  = Decode1                       // dispatcher mode (1 = world/char count) /*0x5ceead*/
//	svrCount = Decode4                    // *((_DWORD*)this+66)                     /*0x5ceeca*/
//	charCount= Decode4                    // *((_DWORD*)this+67)                     /*0x5ceee1*/
// Atlas CharacterViewAllCount writes [byte code][int worldCount][int unk]; code
// carries the mode byte (1).
//
// packet-audit:verify packet=character/clientbound/CharacterViewAllCount version=gms_v79 ida=0x5cee77
func TestCharacterViewAllCountByteOutputV79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	got := NewCharacterViewAllCount(1, 2, 3).Encode(nil, ctx)(nil)
	want := []byte{
		0x01,                   // mode/code = 1 (case 1u)            /*0x5ceead*/
		0x02, 0x00, 0x00, 0x00, // worldCount (Decode4)              /*0x5ceeca*/
		0x03, 0x00, 0x00, 0x00, // unk (Decode4)                     /*0x5ceee1*/
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v79 CharacterViewAllCount wire: got %x want %x", got, want)
	}
}

// CharacterViewAllSearchFailed v79 byte-fixture — mode 2 (case 2u @0x5ceead):
// after the mode byte the client performs NO further wire reads (it clears the
// VAC state and shows a StringPool notice @0x5cef55). Atlas CharacterViewAllSearchFailed
// writes just [byte code]; code carries the mode byte (2).
//
// packet-audit:verify packet=character/clientbound/CharacterViewAllSearchFailed version=gms_v79 ida=0x5cee77
func TestCharacterViewAllSearchFailedByteOutputV79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	got := NewCharacterViewAllSearchFailed(2).Encode(nil, ctx)(nil)
	want := []byte{
		0x02, // mode/code = 2 (case 2u, no further reads)           /*0x5ceead*/
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v79 CharacterViewAllSearchFailed wire: got %x want %x", got, want)
	}
}

// CharacterViewAllError v79 byte-fixture — default mode (an unhandled mode byte,
// e.g. 8, falls through to the default branch @0x5cf22a which shows an error
// modal and performs NO further wire reads). Atlas CharacterViewAllError writes
// just [byte code]; code carries the mode byte. This mirrors the SearchFailed
// shape and, like v83/v87/v95, pins against the same base function (no distinct
// #CharacterViewAllError export slice exists).
//
// packet-audit:verify packet=character/clientbound/CharacterViewAllError version=gms_v79 ida=0x5cee77
func TestCharacterViewAllErrorByteOutputV79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	got := NewCharacterViewAllError(8).Encode(nil, ctx)(nil)
	want := []byte{
		0x08, // mode/code = 8 (default branch, no further reads)     /*0x5cf22a*/
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v79 CharacterViewAllError wire: got %x want %x", got, want)
	}
}

// ---------------------------------------------------------------------------
// Stage E batch 16 — character part E (spawn/move/damage/expression/chair).
// v79 IDB GMS_v79_1_DEVM.exe @13340. Every remote-user clientbound op has its
// leading Decode4(characterId) consumed by the pool dispatcher
// CUserPool::OnUserRemotePacket @0x8c8d4a (v4 = Decode4(a3) @0x8c8d54) BEFORE
// the per-op body runs; SPAWN_PLAYER's charId is read by
// CUserPool::OnUserEnterField @0x8c8978 (Decode4 @0x8c898c). The fixtures below
// pin that charId prefix plus the op body in client read order.
// ---------------------------------------------------------------------------

// CharacterSpawn v79 byte-fixture — SPAWN_PLAYER, op 149.
//
// Client read order: CUserPool::OnUserEnterField @0x8c8978 reads Decode4
// characterId (@0x8c898c) then calls CUserRemote::Init sub_8D589E @0x8d589e;
// after Init it calls CField::DecodeFieldSpecificData @0x513a15 which forwards
// only the CUser (sub_522F28(a2)), never the packet — so NO trailing team byte.
//
// CUserRemote::Init sub_8D589E read order (v79):
//
//	name        = DecodeStr                 // this+4220           /*0x8d58c9*/  (NO leading level byte)
//	guildName   = DecodeStr                 // this+4224           /*0x8d58f9*/
//	logoBg      = Decode2                   // this+4228           /*0x8d592f*/
//	logoBgColor = Decode1                   // this+4230           /*0x8d593d*/
//	logo        = Decode2                   // this+4232           /*0x8d594a*/
//	logoColor   = Decode1                   // this+4234           /*0x8d5956*/
//	SecondaryStat::DecodeForRemote          // cts (opaque §5)     /*0x8d5969*/
//	jobId       = Decode2                   // this+13312          /*0x8d597a*/
//	AvatarLook::Decode                      // avatar (opaque §5)  /*0x8d59b3*/
//	choco       = Decode4                   // SetCarryItemEffect  /*0x8d59c8*/
//	itemEffect  = Decode4                   // SetActiveEffectItem /*0x8d59d2*/
//	chair       = Decode4                   // SetActivePortableChair /*0x8d59dc*/
//	x           = Decode2                                          /*0x8d59ec*/
//	y           = Decode2                                          /*0x8d59f9*/
//	stance      = Decode1                   // this+1392           /*0x8d5a04*/
//	foothold    = Decode2                                          /*0x8d5a14*/
//	bShowAdmin  = Decode1                                          /*0x8d5a91*/
//	pets while(Decode1) loop, 0 terminator                        /*0x8d5b21*/
//	mountLevel  = Decode4                   // this+10240          /*0x8d5b9a*/
//	mountExp    = Decode4                   // this+10244          /*0x8d5ba7*/
//	mountTired  = Decode4                   // this+10248          /*0x8d5bb4*/
//	miniRoom    = Decode1 (0 => skip)                              /*0x8d5bc9*/
//	adBoard     = Decode1 (0 => skip)                              /*0x8d5d2f*/
//	couple      = Decode1 (0 => skip)                              /*0x8d5e5c*/
//	friend      = Decode1 (0 => skip)                              /*0x8d5ea1*/
//	marriage    = Decode1 (0 => skip)                             /*0x8d5ee6*/
//	newYearCard = Decode1 (0 => skip)                             /*0x8d5f2b*/
//	effectFlag  = Decode1 (last read)                             /*0x8d5f67*/
//
// Legacy divergences vs v83 CUserRemote::Init @0x97f55d: v83 reads a leading
// Decode1 level (@0x97f589), a SECOND trailing effect byte (@0x97fd90) and a
// team byte; v79 has none of the three. The cts/avatar blocks are opaque §5
// VERIFIED-EXCEPTIONs; the fixture pins the deterministic charId+name+guild
// prefix (proving no level byte) and the jobId..effectFlag suffix (proving no
// trailing team byte), skipping the time-dependent cts middle.
//
// packet-audit:verify packet=character/clientbound/CharacterSpawn version=gms_v79 ida=0x8c8978
func TestCharacterSpawnByteOutputV79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	guild := GuildEmblem{Name: "TestGuild", LogoBackground: 1, LogoBackgroundColor: 2, Logo: 3, LogoColor: 4}
	cts := model.NewCharacterTemporaryStat()
	in := NewCharacterSpawn(12345, 50, "TestChar", guild, cts, 100, model.Avatar{}, nil, false, 100, 200, 3)
	got := in.Encode(nil, ctx)(nil)

	// Prefix through the guild emblem — proves NO level byte follows charId
	// (the name-length short 0x08,0x00 sits at offset 4, where v83 has level).
	var wantPrefix []byte
	wantPrefix = append(wantPrefix, 0x39, 0x30, 0x00, 0x00) // characterId 12345 (Decode4) /*0x8c898c*/
	// NO level byte /*(v83 @0x97f589)*/
	wantPrefix = append(wantPrefix, 0x08, 0x00, 0x54, 0x65, 0x73, 0x74, 0x43, 0x68, 0x61, 0x72) // "TestChar" /*0x8d58c9*/
	wantPrefix = append(wantPrefix, 0x09, 0x00, 0x54, 0x65, 0x73, 0x74, 0x47, 0x75, 0x69, 0x6c, 0x64) // "TestGuild" /*0x8d58f9*/
	wantPrefix = append(wantPrefix, 0x01, 0x00) // logoBg (Decode2)      /*0x8d592f*/
	wantPrefix = append(wantPrefix, 0x02)       // logoBgColor (Decode1) /*0x8d593d*/
	wantPrefix = append(wantPrefix, 0x03, 0x00) // logo (Decode2)        /*0x8d594a*/
	wantPrefix = append(wantPrefix, 0x04)       // logoColor (Decode1)   /*0x8d5956*/
	if !bytes.HasPrefix(got, wantPrefix) {
		t.Errorf("v79 CharacterSpawn prefix (no level byte):\n got %x\nwant %x", got[:min(len(got), len(wantPrefix))], wantPrefix)
	}

	// Suffix from jobId — proves NO trailing team byte / no 2nd effect byte
	// (the last wire byte is the single effect flag @0x8d5f67).
	avatarBytes := model.Avatar{}.Encode(nil, ctx)(nil)
	var wantSuffix []byte
	wantSuffix = append(wantSuffix, 0x64, 0x00) // jobId 100 (Decode2)   /*0x8d597a*/
	wantSuffix = append(wantSuffix, avatarBytes...) // avatar (opaque §5) /*0x8d59b3*/
	wantSuffix = append(wantSuffix, 0x00, 0x00, 0x00, 0x00) // choco (Decode4)      /*0x8d59c8*/
	wantSuffix = append(wantSuffix, 0x00, 0x00, 0x00, 0x00) // itemEffect (Decode4) /*0x8d59d2*/
	wantSuffix = append(wantSuffix, 0x00, 0x00, 0x00, 0x00) // chair (Decode4)      /*0x8d59dc*/
	wantSuffix = append(wantSuffix, 0x64, 0x00) // x 100 (Decode2)       /*0x8d59ec*/
	wantSuffix = append(wantSuffix, 0xc8, 0x00) // y 200 (Decode2)       /*0x8d59f9*/
	wantSuffix = append(wantSuffix, 0x03)       // stance (Decode1)      /*0x8d5a04*/
	wantSuffix = append(wantSuffix, 0x00, 0x00) // foothold (Decode2)    /*0x8d5a14*/
	wantSuffix = append(wantSuffix, 0x00)       // bShowAdmin (Decode1)  /*0x8d5a91*/
	wantSuffix = append(wantSuffix, 0x00)       // pets terminator       /*0x8d5b21*/
	wantSuffix = append(wantSuffix, 0x01, 0x00, 0x00, 0x00) // mountLevel (Decode4) /*0x8d5b9a*/
	wantSuffix = append(wantSuffix, 0x00, 0x00, 0x00, 0x00) // mountExp (Decode4)   /*0x8d5ba7*/
	wantSuffix = append(wantSuffix, 0x00, 0x00, 0x00, 0x00) // mountTired (Decode4) /*0x8d5bb4*/
	wantSuffix = append(wantSuffix, 0x00) // miniRoom (Decode1)    /*0x8d5bc9*/
	wantSuffix = append(wantSuffix, 0x00) // adBoard (Decode1)     /*0x8d5d2f*/
	wantSuffix = append(wantSuffix, 0x00) // couple (Decode1)      /*0x8d5e5c*/
	wantSuffix = append(wantSuffix, 0x00) // friend (Decode1)      /*0x8d5ea1*/
	wantSuffix = append(wantSuffix, 0x00) // marriage (Decode1)    /*0x8d5ee6*/
	wantSuffix = append(wantSuffix, 0x00) // newYearCard (Decode1) /*0x8d5f2b*/
	wantSuffix = append(wantSuffix, 0x00) // effectFlag (Decode1, last read) /*0x8d5f67*/
	if !bytes.HasSuffix(got, wantSuffix) {
		n := len(wantSuffix)
		if n > len(got) {
			n = len(got)
		}
		t.Errorf("v79 CharacterSpawn suffix (no team byte):\n got %x\nwant %x", got[len(got)-n:], wantSuffix)
	}
}

// CharacterMovement v79 byte-fixture — MOVE_PLAYER, op 171.
//
// CUserRemote::OnMove @0x8c8eff is a thunk to CMovePath::OnMovePacket @0x6583fc
// (the opaque move-path block, OPAQUE_LEDGER VERIFIED-EXCEPTION "model.Movement
// shared path"); characterId(4 LE) is read by the pool dispatcher
// CUserPool::OnUserRemotePacket (Decode4 @0x8c8d54) before dispatch. Wire =
// characterId(4) + the independently-audited model.Movement blob.
//
// packet-audit:verify packet=character/clientbound/CharacterMovement version=gms_v79 ida=0x8c8eff
func TestCharacterMovementByteOutputV79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	opts := normalTypesOptions()
	mv := model.Movement{
		StartX: 100, StartY: 200,
		Elements: []model.MovementCodec{
			&model.NormalElement{Element: model.Element{
				ElemType: 0, X: 110, Y: 210, Vx: 5, Vy: -3, Fh: 1, BMoveAction: 7, TElapse: 50,
			}},
		},
	}
	got := NewCharacterMovement(0x01020304, mv).Encode(nil, ctx)(opts)

	if !bytes.Equal(got[:4], []byte{0x04, 0x03, 0x02, 0x01}) { // charId LE /*0x8c8d54*/
		t.Errorf("v79 CharacterMovement charId: got %x want 04030201", got[:4])
	}
	wantBlob := mv.Encode(nil, ctx)(opts) // opaque move-path (CMovePath::OnMovePacket @0x6583fc)
	if !bytes.Equal(got[4:], wantBlob) {
		t.Errorf("v79 CharacterMovement move-path blob:\n got %x\nwant %x", got[4:], wantBlob)
	}
}

// CharacterDamage v79 byte-fixture — DAMAGE_PLAYER, op 178.
//
// CUserRemote::OnHit @0x8d9489 read order (physical, attackIdx = -1, block
// entered since v4 > -2 @0x8d94bf):
//
//	attackIdx        = Decode1                                     /*0x8d94a6*/
//	damage           = Decode4                                     /*0x8d94bc*/
//	monsterTemplate  = Decode4                                     /*0x8d94d0*/
//	left             = Decode1                                     /*0x8d94de*/
//	stance           = Decode1 (0 => inner block skipped)         /*0x8d9638*/
//	stanceRelated    = Decode1                                    /*0x8d9755*/
//	damage (repeat)  = Decode4                                    /*0x8d9781*/
//
// characterId(4) is read by the pool dispatcher (Decode4 @0x8c8d54). No bGuard
// byte (that is GMS>=95 only; v79 < 95). Matches CharacterDamage.Encode <95 path.
//
// packet-audit:verify packet=character/clientbound/CharacterDamage version=gms_v79 ida=0x8d9489
func TestCharacterDamageByteOutputV79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	got := NewCharacterDamage(1234, model.DamageTypePhysical, 500, 100100, true).Encode(nil, ctx)(nil)
	want := []byte{
		0xd2, 0x04, 0x00, 0x00, // characterId 1234 (dispatcher Decode4) /*0x8c8d54*/
		0xff,                   // attackIdx -1 (Decode1)                /*0x8d94a6*/
		0xf4, 0x01, 0x00, 0x00, // damage 500 (Decode4)                  /*0x8d94bc*/
		0x04, 0x87, 0x01, 0x00, // monsterTemplateId 100100 (Decode4)    /*0x8d94d0*/
		0x01,                   // left (Decode1)                        /*0x8d94de*/
		0x00,                   // stance (Decode1)                      /*0x8d9638*/
		0x00,                   // stanceRelated (Decode1)               /*0x8d9755*/
		0xf4, 0x01, 0x00, 0x00, // damage repeated (Decode4)             /*0x8d9781*/
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v79 CharacterDamage wire:\n got %x\nwant %x", got, want)
	}
}

// CharacterExpression v79 byte-fixture — FACIAL_EXPRESSION, op 179.
//
// Read inline in CUserPool::OnUserRemotePacket case 179 @0x8c8dca:
// v6 = Decode4(expressionId), then CAvatar::SetEmotion(RemoteUser+136, v6, -1) —
// NO duration and NO byItemOption (those are GMS>87 / JMS only; v79 < 87).
// characterId(4) is read by the dispatcher (Decode4 @0x8c8d54).
//
// packet-audit:verify packet=character/clientbound/CharacterExpression version=gms_v79 ida=0x44d7ac
func TestCharacterExpressionByteOutputV79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	got := NewCharacterExpression(12345, 5, 3000).Encode(nil, ctx)(nil)
	want := []byte{
		0x39, 0x30, 0x00, 0x00, // characterId 12345 (dispatcher Decode4) /*0x8c8d54*/
		0x05, 0x00, 0x00, 0x00, // expression 5 (Decode4)                 /*0x8c8dca*/
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v79 CharacterExpression wire: got %x want %x", got, want)
	}
}

// CharacterChairShow v79 byte-fixture — SHOW_CHAIR, op 182.
//
// Read inline in CUserPool::OnUserRemotePacket case 182 @0x8c8e35:
// *((_DWORD*)RemoteUser + 3320) = Decode4(chairId); characterId(4) is read by
// the dispatcher (Decode4 @0x8c8d54). Two LE uint32s.
//
// packet-audit:verify packet=character/clientbound/CharacterChairShow version=gms_v79 ida=0x8c8d4a
func TestCharacterChairShowByteOutputV79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	got := NewCharacterChairShow(1234, 3010000).Encode(nil, ctx)(nil)
	want := []byte{
		0xd2, 0x04, 0x00, 0x00, // characterId 1234 (dispatcher Decode4) /*0x8c8d54*/
		0xd0, 0xed, 0x2d, 0x00, // chairId 3010000 (Decode4)             /*0x8c8e35*/
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v79 CharacterChairShow wire: got %x want %x", got, want)
	}
}

// CharacterSitResult v79 byte-fixture — CANCEL_CHAIR clientbound, op 191.
//
// CUserLocal::OnSitResult @0x8b17fe: flag = Decode1 @0x8b1825; if flag != 0 then
// nSeat = Decode2 @0x8b187b (else stand-up branch reads nothing more). One
// fixture per branch mode.
//
// packet-audit:verify packet=character/clientbound/CharacterSitResult version=gms_v79 ida=0x8b17fe
func TestCharacterSitResultByteOutputV79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	// sit: flag(1)=1 then chairId(2, LE) read.
	gotSit := NewCharacterSit(17).Encode(nil, ctx)(nil)
	wantSit := []byte{0x01, 0x11, 0x00} // flag=1 /*0x8b1825*/, chairId 17 (Decode2) /*0x8b187b*/
	if !bytes.Equal(gotSit, wantSit) {
		t.Errorf("v79 CharacterSitResult sit: got %x want %x", gotSit, wantSit)
	}
	// cancel: flag(1)=0, stand-up branch reads nothing else.
	gotCancel := NewCharacterCancelSit().Encode(nil, ctx)(nil)
	wantCancel := []byte{0x00} // flag=0 /*0x8b1825*/
	if !bytes.Equal(gotCancel, wantCancel) {
		t.Errorf("v79 CharacterSitResult cancel: got %x want %x", gotCancel, wantCancel)
	}
}
