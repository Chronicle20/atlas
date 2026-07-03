package clientbound

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// v61 character clientbound byte fixtures (GMS_v61.1_U_DEVM.exe, port 13338).
//
// The AvatarLook wire block is byte-identical to v79 (avatarBlockV79): v61
// AvatarLook::Decode @0x4b76c6 reads gender, skin, face, !mega, hair, equip loop
// (0xFF term), masked loop (0xFF term), cashWeapon, pets DecodeBuffer(12) — the
// same order as v79, and Avatar.Encode only gates on <=28/>28 (v61 is >28).
//
// LEGACY DIVERGENCE vs v72 in GW_CharacterStat: v61 GW_CharacterStat::Decode
// @0x4b4081 (list path a3=0) reads id, name(13), gender, skin, face, hair,
// pets(24), level, 11x short (job..sp), exp(4), fame(2), then mapId (Decode4
// @0x4b424d, SecureTear<unsigned long> = the mapId slot) and spawnPoint
// (Decode1 @0x4b4267) — and RETURNS. There is NO gachaExp int and NO trailing
// int (both entered in (61,72]). character_statistics.go gates gachaExp and the
// trailing int on GMS>61, so statBlockV61 == statBlockV79 minus those 8 bytes.

// statBlockV61 = the GW_CharacterStat wire for heroStatsV79 under a v61 context:
// identical to statBlockV79 up to fame, then mapId + spawnPoint, with NO gachaExp
// and NO trailing int.
var statBlockV61 = []byte{
	0x04, 0x03, 0x02, 0x01, // id (Decode4)                        /*0x4b40a4*/
	0x48, 0x65, 0x72, 0x6f, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // name (DecodeBuffer 13) /*0x4b40b5*/
	0x00,                   // gender (Decode1)                    /*0x4b40cc*/
	0x00,                   // skin (Decode1)                      /*0x4b40e1*/
	0xd2, 0x04, 0x00, 0x00, // face (Decode4)                      /*0x4b40f6*/
	0x7b, 0x00, 0x00, 0x00, // hair (Decode4)                      /*0x4b410b*/
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // pet long 0 (DecodeBuffer 24) /*0x4b4116*/
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // pet long 1
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // pet long 2
	0x0a,       // level (Decode1)                     /*0x4b4125*/
	0x64, 0x00, // job (Decode2)                       /*0x4b4131*/
	0x04, 0x00, // str (Decode2)                       /*0x4b4145*/
	0x05, 0x00, // dex                                 /*0x4b4159*/
	0x06, 0x00, // int                                 /*0x4b416d*/
	0x07, 0x00, // luck                                /*0x4b4181*/
	0x64, 0x00, // hp                                  /*0x4b4195*/
	0x64, 0x00, // maxHp                               /*0x4b41a9*/
	0x32, 0x00, // mp                                  /*0x4b41bd*/
	0x32, 0x00, // maxMp                               /*0x4b41d1*/
	0x03, 0x00, // ap (Decode2)                        /*0x4b41e5*/
	0x02, 0x00, // sp (Decode2)                        /*0x4b41ff*/
	0x00, 0x00, 0x00, 0x00, // exp (Decode4)                       /*0x4b4219*/
	0x08, 0x00, // fame (Decode2)                      /*0x4b4233*/
	// NO gachaExp int (v61 < gacha boundary)
	0xb8, 0x0b, 0x00, 0x00, // mapId (Decode4)                     /*0x4b424d*/
	0x00, // spawnPoint (Decode1)                /*0x4b4267*/
	// NO trailing int (v61)
}

// CharacterList v61 byte-fixture.
//
// Client read order — v61 char-list decoder sub_56688D @0x56688D, world-select
// success path (status 0/12/23, LABEL_32 @0x566a1a):
//
//	status = Decode1                     // result/status byte              /*0x5668e5*/
//	count  = Decode1                     // number of avatar entries         /*0x566a4c*/
//	for each entry (6 slots, count decoded):
//	    GW_CharacterStat::Decode         // @0x4b4081                        /*0x566a72*/
//	    AvatarLook::Decode               // @0x4b76c6                        /*0x566a80*/
//	    rankEnabled = Decode1            // 0 => memset 16; else buffer 16    /*0x566a88*/
//	        if rankEnabled: DecodeBuffer(16) = 4 rank ints                    /*0x566aa3*/
//	slots  = Decode4                     // m_nSlotCount — NO hasPic byte     /*0x566b02*/
//
// As in v72: NO per-entry family byte (v61 < 73) and NO login-option (hasPic)
// byte before slots (v61 < 83). Only the GW_CharacterStat tail differs (v61 has
// no gachaExp / no trailing int) — captured by statBlockV61.
//
// packet-audit:verify packet=character/clientbound/CharacterList version=gms_v61 ida=0x56688d
func TestCharacterListByteOutputV61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	entry := model.NewCharacterListEntry(heroStatsV79(), heroAvatarV79(), false, false, 1, 2, 3, 4)
	got := NewCharacterList(0, []model.CharacterListEntry{entry}, false, 8).Encode(nil, ctx)(nil)

	var want []byte
	want = append(want, 0x00) // status                /*0x5668e5*/
	want = append(want, 0x01) // count = 1              /*0x566a4c*/
	want = append(want, statBlockV61...)
	want = append(want, avatarBlockV79...)
	// NO family byte (v61 < 73) — the next byte is the rank-enabled flag.
	want = append(want, 0x01)         // rankEnabled = !gm     /*0x566a88*/
	want = append(want, rank16V79...) // rank ints             /*0x566aa3*/
	want = append(want, 0x08, 0x00, 0x00, 0x00) // slots — no hasPic /*0x566b02*/

	if !bytes.Equal(got, want) {
		t.Errorf("v61 CharacterList wire:\n got %x\nwant %x", got, want)
	}
}

// CharacterViewAllCharacters v61 byte-fixture.
//
// Client read order — CLogin::OnViewAllCharResult @0x5671B1, mode 0 (per-world
// character batch, case 0u @0x5671e7):
//
//	mode   = Decode1                     // dispatcher mode (0 = char batch)  /*0x5671e7*/
//	worldId= Decode1                     // v24                               /*0x567401*/
//	count  = Decode1                     // v13                               /*0x567411*/
//	for each of count entries:
//	    GW_CharacterStat::Decode         // @0x4b4081                         /*0x567451*/
//	    AvatarLook::Decode               // @0x4b76c6                         /*0x56745f*/
//	    // worldId stored locally (NOT read); NO family byte (viewAll)
//	    rankEnabled = Decode1            // 0 => memset 16; else buffer 16     /*0x567473*/
//	        if rankEnabled: DecodeBuffer(16) = 4 rank ints                     /*0x56748e*/
//
// packet-audit:verify packet=character/clientbound/CharacterViewAllCharacters version=gms_v61 ida=0x5671b1
func TestCharacterViewAllCharactersByteOutputV61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	entry := model.NewCharacterListEntry(heroStatsV79(), heroAvatarV79(), true, false, 1, 2, 3, 4)
	got := NewCharacterViewAllCharacters(0, world.Id(1), []model.CharacterListEntry{entry}).Encode(nil, ctx)(nil)

	var want []byte
	want = append(want, 0x00) // mode/code (case 0)    /*0x5671e7*/
	want = append(want, 0x01) // worldId               /*0x567401*/
	want = append(want, 0x01) // count = 1             /*0x567411*/
	want = append(want, statBlockV61...)
	want = append(want, avatarBlockV79...)
	want = append(want, 0x01)         // rankEnabled = !gm (no family byte) /*0x567473*/
	want = append(want, rank16V79...) // rank ints                         /*0x56748e*/

	if !bytes.Equal(got, want) {
		t.Errorf("v61 CharacterViewAllCharacters wire:\n got %x\nwant %x", got, want)
	}
}

// AddCharacterEntry v61 byte-fixture.
//
// Client read order — v61 create-character result @0x566EAB (dispatch case 14
// @0x565760; the IDB symbol is CLogin::OnDeleteCharacterResult, rotated one step
// off its body — verified by body reading [code][stat][avatar]):
//
//	code = Decode1                       // result code                     /*0x566ec6*/
//	// on success (code 0, empty slot found):
//	GW_CharacterStat::Decode             // @0x4b4081 into empty slot        /*0x566f4e*/
//	AvatarLook::Decode                   // @0x4b76c6                        /*0x566f5c*/
//	// family byte + 16-byte rank buffer are ZEROED locally, NOT read        /*0x566f74*/
//
// Matches add_entry.go legacyAddEntry path (GMS v29..v82): [code][stat][avatar]
// with no entry trailer.
//
// packet-audit:verify packet=character/clientbound/AddCharacterEntry version=gms_v61 ida=0x566eab
func TestAddCharacterEntryByteOutputV61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	entry := model.NewCharacterListEntry(heroStatsV79(), heroAvatarV79(), false, false, 1, 2, 3, 4)
	got := NewAddCharacterEntry(0, entry).Encode(nil, ctx)(nil)

	var want []byte
	want = append(want, 0x00) // code                 /*0x566ec6*/
	want = append(want, statBlockV61...)
	want = append(want, avatarBlockV79...)
	// no entry trailer

	if !bytes.Equal(got, want) {
		t.Errorf("v61 AddCharacterEntry wire:\n got %x\nwant %x", got, want)
	}
}

// DeleteCharacterResponse v61 byte-fixture.
//
// Client read order — v61 delete-character result @0x566C86 (dispatch case 15
// @0x565760... case 15 @0x565722; the IDB symbol is CLogin::OnCheckDuplicatedIDResult,
// rotated — verified by body reading [charId][code]):
//
//	characterId = Decode4                // id of the char to remove        /*0x566caa*/
//	code        = Decode1                // result code                     /*0x566cad*/
//
// Matches DeleteCharacterResponse.Encode exactly ([int charId][byte code]).
//
// packet-audit:verify packet=character/clientbound/DeleteCharacterResponse version=gms_v61 ida=0x566c86
func TestDeleteCharacterResponseByteOutputV61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	got := NewDeleteCharacterResponse(12345, 0).Encode(nil, ctx)(nil)
	want := []byte{
		0x39, 0x30, 0x00, 0x00, // characterId = 12345 (Decode4) /*0x566caa*/
		0x00, // code (Decode1)                                  /*0x566cad*/
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v61 DeleteCharacterResponse wire: got %x want %x", got, want)
	}
}

// CharacterNameResponse v61 byte-fixture.
//
// Client read order — v61 name-check result @0x566BAB (dispatch case 13
// @0x56577a; the IDB symbol is CLogin::OnCreateNewCharacterResult, rotated —
// verified by body reading [name string][code]):
//
//	name = DecodeStr                     // echoed character name           /*0x566bc7*/
//	code = Decode1                       // availability/result code        /*0x566bd2*/
//
// Matches CharacterNameResponse.Encode exactly ([string name][byte code]).
//
// packet-audit:verify packet=character/clientbound/CharacterNameResponse version=gms_v61 ida=0x566bab
func TestCharacterNameResponseByteOutputV61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	got := NewCharacterNameResponse("TestChar", 0).Encode(nil, ctx)(nil)
	want := []byte{
		0x08, 0x00, // name length = 8 (DecodeStr)               /*0x566bc7*/
		0x54, 0x65, 0x73, 0x74, 0x43, 0x68, 0x61, 0x72, // "TestChar"
		0x00, // code (Decode1)                                  /*0x566bd2*/
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v61 CharacterNameResponse wire: got %x want %x", got, want)
	}
}

// StatusMessageCashItemExpire v61 byte-fixture — SHOW_STATUS_INFO, op 36.
//
// Client read order — CWvsContext::OnMessage @0x8437EF reads Decode1(nType)
// @0x8437fe then dispatches; the v61 table has arms 0-9 ONLY (SHRUNK from v72's
// 0-11). Case 2 (CASH_ITEM_EXPIRE) → sub_843FBF @0x843fbf:
//
//	nType  = Decode1                     // OnMessage dispatch byte (mode)  /*0x8437fe*/
//	itemId = Decode4                     // CItemInfo::GetItemName(itemId)  /*0x843fce*/
//
// The arm reads exactly one Decode4(itemId) then formats the "expired" string.
// Body has no version gate → byte-identical to v72.
//
// packet-audit:verify packet=character/clientbound/StatusMessageCashItemExpire version=gms_v61 ida=0x843fbf
func TestStatusMessageCashItemExpireByteOutputV61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	got := NewStatusMessageCashItemExpire(2, 5000000).Encode(nil, ctx)(nil)
	want := []byte{
		0x02,                   // mode = 2 (CASH_ITEM_EXPIRE dispatch byte) /*0x8437fe*/
		0x40, 0x4b, 0x4c, 0x00, // itemId 5000000 (Decode4)                 /*0x843fce*/
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v61 StatusMessageCashItemExpire wire: got %x want %x", got, want)
	}
}

// CharacterAppearanceUpdate v61 byte-fixture — UPDATE_CHAR_LOOK, op 152.
//
// Client read order — CUserRemote::OnAvatarModified @0x7CBD86 (characterId is
// read by the pool dispatcher before the body):
//
//	flags = Decode1                  // bitfield: &1 look, &2 speed, &4 carry  /*0x7cbd9f*/
//	if flags & 1:  AvatarLook::Decode  // full avatar/look block (@0x4b76c6)    /*0x7cbda8*/
//	if flags & 2:  Decode1            // riding/vehicle speed (not set here)     /*0x7cbdf9*/
//	if flags & 4:  Decode1            // carry-item effect (not set here)        /*0x7cbe1b*/
//	coupleMarker   = Decode1         // 0 => none; else 2xDecodeBuffer(8)+Decode4 /*0x7cbe2d*/
//	friendMarker   = Decode1         // 0 => none; else 2xDecodeBuffer(8)+Decode4 /*0x7cbe7a*/
//	marriageMarker = Decode1         // != 0 => 3x Decode4; else zeros          /*0x7cbec7*/
//
// As in v72: the marriage if/else has NO trailing unconditional Decode4. Atlas
// always writes flags=1 (look-only) + three ring markers as 0, so the trailing
// WriteInt(0) is benign trailing slack. Byte-identical to the v72 fixture.
//
// packet-audit:verify packet=character/clientbound/CharacterAppearanceUpdate version=gms_v61 ida=0x7cbd86
func TestCharacterAppearanceUpdateByteOutputV61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	avatar := model.NewAvatar(
		1,     // gender
		2,     // skinColor
		0x14,  // face
		false, // mega -> WriteBool(!mega)=WriteBool(true)=0x01
		0x1E,  // hair
		nil, nil, nil,
	)
	got := NewCharacterAppearanceUpdate(0x12345678, avatar).Encode(nil, ctx)(nil)

	want := []byte{
		0x78, 0x56, 0x34, 0x12, // characterId (WriteInt)                 /*dispatch*/
		0x01,                   // flags = 1 (look-only)                  /*0x7cbd9f*/
		// --- AvatarLook block (flags & 1) --- @0x4b76c6                  /*0x7cbda8*/
		0x01,                   // gender (Decode1)
		0x02,                   // skinColor (Decode1)
		0x14, 0x00, 0x00, 0x00, // face (Decode4)
		0x01,                   // !mega -> WriteBool(true) (Decode1)
		0x1e, 0x00, 0x00, 0x00, // hair (Decode4)
		0xff,                   // equipment terminator
		0xff,                   // masked-equipment terminator
		0x00, 0x00, 0x00, 0x00, // cash weapon (Decode4)
		0x00, 0x00, 0x00, 0x00, // pet 0 (DecodeBuffer 12)
		0x00, 0x00, 0x00, 0x00, // pet 1
		0x00, 0x00, 0x00, 0x00, // pet 2
		// --- ring markers ---
		0x00,                   // couple marker (Decode1)                /*0x7cbe2d*/
		0x00,                   // friend marker (Decode1)                /*0x7cbe7a*/
		0x00,                   // marriage marker (Decode1)              /*0x7cbec7*/
		0x00, 0x00, 0x00, 0x00, // completed set item id (trailing slack; unread when marriage==0)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v61 CharacterAppearanceUpdate wire:\n got %x\nwant %x", got, want)
	}
}

// CharacterInfo v61 byte-fixture — CHAR_INFO, op 58.
//
// Client read order — CWvsContext::OnCharacterInfo @0x8455ED:
//
//	Decode4(charId) /*0x845622*/, Decode1(level) /*0x845649*/, Decode2(job) /*0x845653*/,
//	Decode2(fame) /*0x84565d*/, Decode1(married) /*0x84566d*/, DecodeStr(guild) /*0x845674*/,
//	DecodeStr(alliance) /*0x845683*/, Decode1(medalInfo byte) /*0x845698*/,
//	Decode1(first pet flag) /*0x84569b*/ → pet loop sub_763059 @0x763059 (per pet:
//	  Decode4(templateId), DecodeStr(name), Decode1(level), Decode2(closeness),
//	  Decode1(fullness), Decode2(skill), Decode4(itemId), Decode1(next flag) — bool-term),
//	Decode1(mount flag) /*0x845713*/ + 3×Decode4 /*0x845728..*/ → SetTamingMobInfo,
//	Decode1(wish count) /*0x845754*/ + count×Decode4 (DecodeBuffer 4*n) /*0x845780*/,
//	sub_5DD5A3 @0x5dd5a3 monster-book (5×Decode4 @0x5dd5bf..0x5dd5e8) then RETURNS.
//
// LEGACY DIVERGENCE vs v72: v61 reads NO trailing medal block. v72's monster-book
// helper continues into medalId(Decode4)+questCount(Decode2); v61's sub_5DD5A3
// stops after the 5 monster-book ints, and the following helpers (sub_5DEE5F,
// sub_6E8E94) take a local struct, not the packet. info.go gates the medal block
// on GMS>61, so the v61 wire ends at the monster-book cover. NO trailing chair int
// either (>=87 branch absent; v61 < 87).
//
// packet-audit:verify packet=character/clientbound/CharacterInfo version=gms_v61 ida=0x8455ed
func TestCharacterInfoByteOutputV61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	pets := []InfoPet{{Slot: 0, TemplateId: 5000000, Name: "Kitty", Level: 15, Closeness: 200, Fullness: 80}}
	mb := MonsterBookInfo{Level: 5, NormalCards: 10, SpecialCards: 3, TotalCards: 13, Cover: 2380001}
	mount := MountInfo{Active: true, Level: 7, Exp: 1234, Tiredness: 42}
	in := NewCharacterInfo(12345, 50, 100, 10, "TestGuild", pets, []uint32{1002000, 1002001}, 1142007, mb, mount)

	got := in.Encode(nil, ctx)(nil)
	// == v72 golden (v72_test.go TestCharacterInfoByteOutputV72) MINUS the trailing
	// 6 medal bytes (medalId 0x116cf7 + questCount 0). Every prefix byte traces to the
	// same read as v72; v61 omits only the medal block (verified @0x5dd5a3 returns).
	want, _ := hex.DecodeString(
		"393000003264000a00000900546573744775696c6400000001404b4c0005004b697474790fc80050000000000000000107000000d20400002a00000002104a0f00114a0f00050000000a000000030000000d000000e1502400")
	if !bytes.Equal(got, want) {
		t.Errorf("v61 CharacterInfo wire (len got=%d want=%d):\n got %x\nwant %x", len(got), len(want), got, want)
	}
}

// CharacterViewAllCount v61 — mode 1 (case 1u @0x5671e7):
//	mode     = Decode1                     // dispatcher mode (1)             /*0x5671e7*/
//	worldCount = Decode4                   // *((_DWORD*)this+64)             /*0x567204*/
//	unk        = Decode4                   // *((_DWORD*)this+65)             /*0x56721b*/
//
// packet-audit:verify packet=character/clientbound/CharacterViewAllCount version=gms_v61 ida=0x5671b1
func TestCharacterViewAllCountByteOutputV61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	got := NewCharacterViewAllCount(1, 2, 3).Encode(nil, ctx)(nil)
	want := []byte{
		0x01,                   // mode/code = 1 (case 1u) /*0x5671e7*/
		0x02, 0x00, 0x00, 0x00, // worldCount (Decode4)   /*0x567204*/
		0x03, 0x00, 0x00, 0x00, // unk (Decode4)          /*0x56721b*/
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v61 CharacterViewAllCount wire: got %x want %x", got, want)
	}
}

// CharacterViewAllSearchFailed v61 — mode 2 (case 2u @0x5671e7): after the mode
// byte the client performs NO further wire reads (ResetVAC + shows a notice).
//
// packet-audit:verify packet=character/clientbound/CharacterViewAllSearchFailed version=gms_v61 ida=0x5671b1
func TestCharacterViewAllSearchFailedByteOutputV61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	got := NewCharacterViewAllSearchFailed(2).Encode(nil, ctx)(nil)
	want := []byte{0x02} // mode/code = 2 (case 2u, no further reads) /*0x5671e7*/
	if !bytes.Equal(got, want) {
		t.Errorf("v61 CharacterViewAllSearchFailed wire: got %x want %x", got, want)
	}
}

// CharacterViewAllError v61 — default mode (an unhandled mode byte falls through
// to the default branch which shows an ERROR modal and performs NO further wire
// reads).
//
// packet-audit:verify packet=character/clientbound/CharacterViewAllError version=gms_v61 ida=0x5671b1
func TestCharacterViewAllErrorByteOutputV61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	got := NewCharacterViewAllError(8).Encode(nil, ctx)(nil)
	want := []byte{0x08} // mode/code = 8 (default branch, no further reads) /*0x5671e7*/
	if !bytes.Equal(got, want) {
		t.Errorf("v61 CharacterViewAllError wire: got %x want %x", got, want)
	}
}

// AddCharacterError v61 — ADD_NEW_CHAR_ENTRY error path (op 14). The v61 add
// handler @0x566EAB reads Decode1(code) @0x566ec6; a non-zero code branches to an
// error dialog (YesNo2) with no stat/avatar body. AddCharacterError.Encode writes
// the single [code] byte. == v72.
//
// packet-audit:verify packet=character/clientbound/AddCharacterError version=gms_v61 ida=0x566eab
func TestAddCharacterErrorByteOutputV61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	got := NewAddCharacterError(9).Encode(nil, ctx)(nil)
	want := []byte{0x09} // code (Decode1) /*0x566ec6*/
	if !bytes.Equal(got, want) {
		t.Errorf("v61 AddCharacterError wire: got %x want %x", got, want)
	}
}

// ChalkboardUse v61 byte-fixture — CHALKBOARD, op 123.
//
// Client read order — CUser::OnADBoard @0x7912BB (characterId is read by the
// pool dispatcher CUserPool::OnUserCommonPacket before the body):
//
//	active  = Decode1                    // if-guard (0 => clear, no message)
//	message = DecodeStr                  // only when active
//
// No version gate below 72; byte-identical to the v72 fixture.
//
// packet-audit:verify packet=character/clientbound/ChalkboardUse version=gms_v61 ida=0x7912bb
func TestChalkboardUseByteOutputV61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)

	t.Run("active", func(t *testing.T) {
		got := NewChalkboardUse(1234, "Hi").Encode(nil, ctx)(nil)
		want := []byte{
			0xD2, 0x04, 0x00, 0x00, // characterId (dispatcher prefix)
			0x01,       // active = 1 (Decode1)
			0x02, 0x00, // message len = 2 (DecodeStr)
			0x48, 0x69, // "Hi"
		}
		if !bytes.Equal(got, want) {
			t.Errorf("v61 ChalkboardUse active:\n got %x\nwant %x", got, want)
		}
	})

	t.Run("clear", func(t *testing.T) {
		got := NewChalkboardClear(1234).Encode(nil, ctx)(nil)
		want := []byte{0xD2, 0x04, 0x00, 0x00, 0x00} // characterId + active=0
		if !bytes.Equal(got, want) {
			t.Errorf("v61 ChalkboardUse clear:\n got %x\nwant %x", got, want)
		}
	})
}

// CharacterDamage v61 byte-fixture — DAMAGE_PLAYER, op 148.
//
// Client read order — CUserRemote::OnHit @0x7CB9FF (export calls: Decode1,
// Decode4, Decode4, Decode1, Decode1, Decode1(guard), Decode4):
//
//	attackIdx        = Decode1
//	damage           = Decode4
//	monsterTemplate  = Decode4
//	left             = Decode1
//	stance           = Decode1
//	stanceRelated    = Decode1
//	damage (repeat)  = Decode4
//
// characterId(4) read by the pool dispatcher. No bGuard byte (GMS>=95 only; v61
// < 95). Byte-identical to the v72 fixture.
//
// packet-audit:verify packet=character/clientbound/CharacterDamage version=gms_v61 ida=0x7cb9ff
func TestCharacterDamageByteOutputV61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	got := NewCharacterDamage(1234, model.DamageTypePhysical, 500, 100100, true).Encode(nil, ctx)(nil)
	want := []byte{
		0xd2, 0x04, 0x00, 0x00, // characterId 1234 (dispatcher Decode4)
		0xff,                   // attackIdx -1 (Decode1)
		0xf4, 0x01, 0x00, 0x00, // damage 500 (Decode4)
		0x04, 0x87, 0x01, 0x00, // monsterTemplateId 100100 (Decode4)
		0x01,                   // left (Decode1)
		0x00,                   // stance (Decode1)
		0x00,                   // stanceRelated (Decode1)
		0xf4, 0x01, 0x00, 0x00, // damage repeated (Decode4)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v61 CharacterDamage wire:\n got %x\nwant %x", got, want)
	}
}

// CharacterSitResult v61 byte-fixture — CANCEL_CHAIR clientbound, op 160.
//
// Client read order — CUserLocal::OnSitResult @0x7AB9D9: flag = Decode1; if
// flag != 0 then nSeat = Decode2 (else stand-up branch reads nothing more).
// Byte-identical to the v72 fixture.
//
// packet-audit:verify packet=character/clientbound/CharacterSitResult version=gms_v61 ida=0x7ab9d9
func TestCharacterSitResultByteOutputV61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	gotSit := NewCharacterSit(17).Encode(nil, ctx)(nil)
	wantSit := []byte{0x01, 0x11, 0x00} // flag=1, chairId 17 (Decode2)
	if !bytes.Equal(gotSit, wantSit) {
		t.Errorf("v61 CharacterSitResult sit: got %x want %x", gotSit, wantSit)
	}
	gotCancel := NewCharacterCancelSit().Encode(nil, ctx)(nil)
	wantCancel := []byte{0x00} // flag=0
	if !bytes.Equal(gotCancel, wantCancel) {
		t.Errorf("v61 CharacterSitResult cancel: got %x want %x", gotCancel, wantCancel)
	}
}

// CharacterKeyMap v61 byte-fixture — KEYMAP, op 262.
//
// Client read order — CFuncKeyMappedMan::OnInit @0x51AA92: reset = Decode1; if
// reset == 0 the client loops 89 FUNCKEY_MAPPED::Decode entries (each
// DecodeBuffer(5) = nType[1]+nID[4]). v61 reads 89 entries (== v72/v79, NOT the
// 90 the v83 codec emits); keymap.go gates the count to 89 for GMS < 83.
//
// packet-audit:verify packet=character/clientbound/CharacterKeyMap version=gms_v61 ida=0x51aa92
func TestCharacterKeyMapByteOutputV61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	keys := map[int32]KeyBinding{
		2:  {KeyType: 4, KeyAction: 10},
		16: {KeyType: 4, KeyAction: 8},
		41: {KeyType: 4, KeyAction: 11},
	}
	got := NewCharacterKeyMap(keys).Encode(nil, ctx)(nil)

	var want []byte
	want = append(want, 0x00) // not-reset flag (Decode1 == 0)
	for i := int32(0); i < 89; i++ { // v61 reads 89 FUNCKEY_MAPPED entries
		if k, ok := keys[i]; ok {
			want = append(want, byte(k.KeyType))
			want = append(want, byte(k.KeyAction), byte(k.KeyAction>>8), byte(k.KeyAction>>16), byte(k.KeyAction>>24))
		} else {
			want = append(want, 0x00, 0x00, 0x00, 0x00, 0x00)
		}
	}
	if len(got) != 1+89*5 {
		t.Fatalf("v61 CharacterKeyMap length: got %d, want %d", len(got), 1+89*5)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v61 CharacterKeyMap wire:\n got %x\nwant %x", got, want)
	}

	gotReset := NewCharacterKeyMapResetToDefault().Encode(nil, ctx)(nil)
	if !bytes.Equal(gotReset, []byte{0x01}) {
		t.Errorf("v61 CharacterKeyMap reset: got %x want 01", gotReset)
	}
}

// CharacterSpawn v61 byte-fixture — SPAWN_PLAYER, op 120.
//
// Client read order — CUserPool::OnUserEnterField @0x7BD862. As in v72/v79 (all
// GMS < 83): NO level byte after characterId, and NO trailing team byte after
// the effectFlag (both are >=83 additions). spawn.go gates those on the version.
// The avatar/temporary-stat block is the shared opaque model. Byte-identical to
// the v72 fixture (op differs — Δ-25 — but op is not part of the body).
//
// packet-audit:verify packet=character/clientbound/CharacterSpawn version=gms_v61 ida=0x7bd862
func TestCharacterSpawnByteOutputV61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	guild := GuildEmblem{Name: "TestGuild", LogoBackground: 1, LogoBackgroundColor: 2, Logo: 3, LogoColor: 4}
	cts := model.NewCharacterTemporaryStat()
	in := NewCharacterSpawn(12345, 50, "TestChar", guild, cts, 100, model.Avatar{}, nil, false, 100, 200, 3)
	got := in.Encode(nil, ctx)(nil)

	// Prefix through the guild emblem — proves NO level byte follows charId.
	var wantPrefix []byte
	wantPrefix = append(wantPrefix, 0x39, 0x30, 0x00, 0x00) // characterId 12345 (Decode4)
	wantPrefix = append(wantPrefix, 0x08, 0x00, 0x54, 0x65, 0x73, 0x74, 0x43, 0x68, 0x61, 0x72)       // "TestChar"
	wantPrefix = append(wantPrefix, 0x09, 0x00, 0x54, 0x65, 0x73, 0x74, 0x47, 0x75, 0x69, 0x6c, 0x64) // "TestGuild"
	wantPrefix = append(wantPrefix, 0x01, 0x00) // logoBg (Decode2)
	wantPrefix = append(wantPrefix, 0x02)       // logoBgColor (Decode1)
	wantPrefix = append(wantPrefix, 0x03, 0x00) // logo (Decode2)
	wantPrefix = append(wantPrefix, 0x04)       // logoColor (Decode1)
	if !bytes.HasPrefix(got, wantPrefix) {
		n := len(wantPrefix)
		if n > len(got) {
			n = len(got)
		}
		t.Errorf("v61 CharacterSpawn prefix (no level byte):\n got %x\nwant %x", got[:n], wantPrefix)
	}

	// Suffix from jobId — proves NO trailing team byte (last wire byte is effectFlag).
	avatarBytes := model.Avatar{}.Encode(nil, ctx)(nil)
	var wantSuffix []byte
	wantSuffix = append(wantSuffix, 0x64, 0x00) // jobId 100 (Decode2)
	wantSuffix = append(wantSuffix, avatarBytes...)
	wantSuffix = append(wantSuffix, 0x00, 0x00, 0x00, 0x00) // choco (Decode4)
	wantSuffix = append(wantSuffix, 0x00, 0x00, 0x00, 0x00) // itemEffect (Decode4)
	wantSuffix = append(wantSuffix, 0x00, 0x00, 0x00, 0x00) // chair (Decode4)
	wantSuffix = append(wantSuffix, 0x64, 0x00) // x 100 (Decode2)
	wantSuffix = append(wantSuffix, 0xc8, 0x00) // y 200 (Decode2)
	wantSuffix = append(wantSuffix, 0x03)       // stance (Decode1)
	wantSuffix = append(wantSuffix, 0x00, 0x00) // foothold (Decode2)
	wantSuffix = append(wantSuffix, 0x00)       // bShowAdmin (Decode1)
	wantSuffix = append(wantSuffix, 0x00)       // pets terminator
	wantSuffix = append(wantSuffix, 0x01, 0x00, 0x00, 0x00) // mountLevel (Decode4)
	wantSuffix = append(wantSuffix, 0x00, 0x00, 0x00, 0x00) // mountExp (Decode4)
	wantSuffix = append(wantSuffix, 0x00, 0x00, 0x00, 0x00) // mountTired (Decode4)
	wantSuffix = append(wantSuffix, 0x00) // miniRoom (Decode1)
	wantSuffix = append(wantSuffix, 0x00) // adBoard (Decode1)
	wantSuffix = append(wantSuffix, 0x00) // couple (Decode1)
	wantSuffix = append(wantSuffix, 0x00) // friend (Decode1)
	wantSuffix = append(wantSuffix, 0x00) // marriage (Decode1)
	wantSuffix = append(wantSuffix, 0x00) // newYearCard (Decode1)
	wantSuffix = append(wantSuffix, 0x00) // effectFlag (Decode1, last read)
	if !bytes.HasSuffix(got, wantSuffix) {
		n := len(wantSuffix)
		if n > len(got) {
			n = len(got)
		}
		t.Errorf("v61 CharacterSpawn suffix (no team byte):\n got %x\nwant %x", got[len(got)-n:], wantSuffix)
	}
}

// ItemUpgrade v61 byte-fixture — SHOW_SCROLL_EFFECT, op 126.
//
// Client read order — CUser::ShowItemUpgradeEffect @0x78DC86 reads, after the
// pool dispatcher's Decode4 characterId, exactly four Decode1 flags:
//
//	success         = Decode1  /*0x78dcbb*/
//	cursed          = Decode1  /*0x78dcc5*/
//	legendarySpirit = Decode1  /*0x78dcd6*/
//	whiteScroll     = Decode1  /*0x78dce1*/
//
// v61 (GMS < 87) reads NO Decode4 enchantCategory and NO 5th Decode1
// enchantResultFlag (both are GMS>87 / JMS additions). Byte-identical to v72.
//
// packet-audit:verify packet=character/clientbound/ItemUpgrade version=gms_v61 ida=0x78dc86
func TestItemUpgradeByteOutputV61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	got := NewItemUpgrade(12345, true, false, true, false).Encode(nil, ctx)(nil)
	want := []byte{
		0x39, 0x30, 0x00, 0x00, // characterId 12345 (dispatcher Decode4)
		0x01, // success = true (Decode1)         /*0x78dcbb*/
		0x00, // cursed = false (Decode1)         /*0x78dcc5*/
		0x01, // legendarySpirit = true (Decode1) /*0x78dcd6*/
		0x00, // whiteScroll = false (Decode1)    /*0x78dce1*/
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v61 ItemUpgrade wire:\n got %x\nwant %x", got, want)
	}
}

// EffectSimple v61 byte-fixture — CUser::OnEffect mode-only arms.
//
// CUser::OnEffect @0x79148D dispatches on Decode1(mode); case 0 (LevelUp,
// @0x7914b0) reads ONLY that mode byte and plays a client-side quest effect —
// no further wire fields. EffectSimple.Encode writes exactly that byte. Shares the
// OnEffect demux with EffectQuest/EffectSkillUse (SHOW_FOREIGN_EFFECT /
// SHOW_ITEM_GAIN_INCHAT grade worst-of all three).
//
// packet-audit:verify packet=character/clientbound/EffectSimple version=gms_v61 ida=0x79148d
func TestEffectSimpleByteOutputV61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	gotSelf := NewEffectSimple(0).Encode(nil, ctx)(nil)
	if want := []byte{0x00}; !bytes.Equal(gotSelf, want) {
		t.Errorf("v61 EffectSimple self: got %x want %x", gotSelf, want)
	}
	gotForeign := NewEffectSimpleForeign(0x12345678, 8).Encode(nil, ctx)(nil)
	if want := []byte{0x78, 0x56, 0x34, 0x12, 0x08}; !bytes.Equal(gotForeign, want) {
		t.Errorf("v61 EffectSimpleForeign: got %x want %x", gotForeign, want)
	}
}

// EffectSkillUse v61 byte-fixture — CUser::OnEffect case 1 (SHOW_SKILL_USE_EFFECT).
//
// CUser::OnEffect @0x79148D case 1u (@0x791570) reads: Decode4(skillId) @0x79157a
// then Decode1(skillLevel) @0x79158e (fed to SKILLENTRY::IsActionAppointed). v61
// (GMS < 83) reads NO caster-level byte (introduced at v83); effect_skill_use.go
// gates it via effectSkillUseIncludesCharacterLevel. mode(1)+skillId(4)+skillLevel(1).
//
// packet-audit:verify packet=character/clientbound/EffectSkillUse version=gms_v61 ida=0x79148d
func TestEffectSkillUseByteOutputV61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	got := NewEffectSkillUse(1, 1001004, 0, 5, false, false, false, false, false, false).Encode(nil, ctx)(nil)
	want := []byte{
		0x01,                   // mode = 1 (Decode1 selector)
		0x2c, 0x46, 0x0f, 0x00, // skillId 1001004 (Decode4)   /*0x79157a*/
		// NO caster-level byte (v61 < 83)
		0x05, // skillLevel = 5 (Decode1)                       /*0x79158e*/
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v61 EffectSkillUse wire:\n got %x\nwant %x", got, want)
	}
}

// EffectQuest v61 byte-fixture — SHOW_FOREIGN_EFFECT (153) / SHOW_ITEM_GAIN_INCHAT (161).
//
// CUser::OnEffect @0x79148D case 3u (@0x791820) reads:
//
//	count = Decode1                        /*0x79182c*/
//	if count == 0:
//	    message = DecodeStr                /*0x791993*/
//	    nEffect = Decode4                  /*0x7919ca*/
//	else (count>0): repeat count times:
//	    itemId  = Decode4                  /*0x79184f*/
//	    amount  = Decode4                  /*0x791860*/
//
// mode byte (case 3) precedes. Byte-identical to v72.
//
// packet-audit:verify packet=character/clientbound/EffectQuest version=gms_v61 ida=0x79148d
func TestEffectQuestByteOutputV61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)

	t.Run("message", func(t *testing.T) {
		got := NewEffectQuest(3, "Hello", 0x10, nil).Encode(nil, ctx)(nil)
		want := []byte{
			0x03,                                     // mode (OnEffect case 3)
			0x00,                                     // count = 0 (Decode1)
			0x05, 0x00, 0x48, 0x65, 0x6c, 0x6c, 0x6f, // "Hello" (DecodeStr)
			0x10, 0x00, 0x00, 0x00, // nEffect 0x10 (Decode4)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("v61 EffectQuest message:\n got %x\nwant %x", got, want)
		}
	})

	t.Run("rewards", func(t *testing.T) {
		got := NewEffectQuest(3, "", 0, []QuestReward{{ItemId: 2000000, Amount: 5}}).Encode(nil, ctx)(nil)
		want := []byte{
			0x03,                   // mode
			0x01,                   // count = 1 (Decode1)
			0x80, 0x84, 0x1e, 0x00, // itemId 2000000 (Decode4)
			0x05, 0x00, 0x00, 0x00, // amount 5 (Decode4)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("v61 EffectQuest rewards:\n got %x\nwant %x", got, want)
		}
	})
}

// CharacterExpression v61 byte-fixture — FACIAL_EXPRESSION, op 149.
//
// Read inline in CUserPool::OnUserRemotePacket case 149: v6 = Decode4(expression),
// then CAvatar::SetEmotion(RemoteUser, v6, -1) — NO duration, NO byItemOption
// (GMS>87 / JMS only; v61 < 87). characterId(4) is read by the dispatcher.
// Byte-identical to the v72 fixture.
//
// packet-audit:verify packet=character/clientbound/CharacterExpression version=gms_v61 ida=0x43e877
func TestCharacterExpressionByteOutputV61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	got := NewCharacterExpression(12345, 5, 3000).Encode(nil, ctx)(nil)
	want := []byte{
		0x39, 0x30, 0x00, 0x00, // characterId 12345 (dispatcher Decode4)
		0x05, 0x00, 0x00, 0x00, // expression 5 (Decode4)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v61 CharacterExpression wire: got %x want %x", got, want)
	}
}

// CharacterChairShow v61 byte-fixture — SHOW_CHAIR, op 151.
//
// Read inline in CUserPool::OnUserRemotePacket case 151 (RemoteUser+2546 =
// Decode4(chairId), CUserRemote::OnSetActivePortableChair @0x7BDBDA); characterId(4)
// is read by the dispatcher. Two LE uint32s. Byte-identical to the v72 fixture.
//
// packet-audit:verify packet=character/clientbound/CharacterChairShow version=gms_v61 ida=0x7bdbda
func TestCharacterChairShowByteOutputV61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	got := NewCharacterChairShow(1234, 3010000).Encode(nil, ctx)(nil)
	want := []byte{
		0xd2, 0x04, 0x00, 0x00, // characterId 1234 (dispatcher Decode4)
		0xd0, 0xed, 0x2d, 0x00, // chairId 3010000 (Decode4)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v61 CharacterChairShow wire: got %x want %x", got, want)
	}
}
