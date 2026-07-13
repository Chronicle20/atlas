package clientbound

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// v72 character clientbound byte fixtures (GMS_v72.1_U_DEVM.exe, port 13339).
//
// The GW_CharacterStat and AvatarLook wire blocks are byte-identical to the v79
// fixture (v79_test.go): v72 shares every relevant GW_CharacterStat gate (>28 pet
// longs + gachaExp, <95 int16 HP/MP, >12 trailing int, >=87 false → no nSubJob),
// so statBlockV79 / avatarBlockV79 / rank16V79 are reused verbatim here.
//
// GW_CharacterStat::Decode (v72 @0x4cf0ee) and AvatarLook::Decode (v72 @0x4d340d)
// are the shared decoders the char-mgmt handlers below call.
//
// LEGACY DIVERGENCE vs v79: the per-entry family/viewAll placeholder byte is
// ABSENT in v72 (introduced at GMS v73). The v72 char-list loop (sub_5B3646
// @0x5b384d) reads only ONE Decode1 (the rank-enabled flag) after the avatar
// block; v79 reads TWO (family @0x5ce743 + rankEnabled @0x5ce745).
// character_list_entry.go gates the family byte on GMS>=73.

// CharacterList v72 byte-fixture.
//
// Client read order — v72 char-list decoder sub_5B3646 @0x5B3646, world-select
// success path (status 0/12/23, LABEL_32 @0x5b37d7):
//
//	status = Decode1                     // result/status byte              /*0x5b369e*/
//	count  = Decode1                     // number of avatar entries         /*0x5b3808*/
//	for each entry (6 slots, count decoded):
//	    GW_CharacterStat::Decode         // @0x4cf0ee                        /*0x5b3837*/
//	    AvatarLook::Decode               // @0x4d340d                        /*0x5b3845*/
//	    rankEnabled = Decode1            // 0 => memset 16; else buffer 16    /*0x5b384d*/
//	        if rankEnabled: DecodeBuffer(16) = 4 rank ints                    /*0x5b3868*/
//	slots  = Decode4                     // m_nSlotCount — NO hasPic byte     /*0x5b38ba*/
//
// Two legacy divergences vs v83+: (1) NO per-entry family byte (v72 < 73), and
// (2) NO login-option (hasPic) byte before slots (v72 < 83; list.go skips it).
//
// packet-audit:verify packet=character/clientbound/CharacterList version=gms_v72 ida=0x5b3646
func TestCharacterListByteOutputV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	entry := model.NewCharacterListEntry(heroStatsV79(), heroAvatarV79(), false, false, 1, 2, 3, 4)
	got := NewCharacterList(0, []model.CharacterListEntry{entry}, false, 8).Encode(nil, ctx)(nil)

	var want []byte
	want = append(want, 0x00) // status                /*0x5b369e*/
	want = append(want, 0x01) // count = 1              /*0x5b3808*/
	want = append(want, statBlockV79...)
	want = append(want, avatarBlockV79...)
	// NO family byte (v72 < 73) — the next byte is the rank-enabled flag.
	want = append(want, 0x01)         // rankEnabled = !gm     /*0x5b384d*/
	want = append(want, rank16V79...) // rank ints             /*0x5b3868*/
	want = append(want, 0x08, 0x00, 0x00, 0x00) // slots — no hasPic /*0x5b38ba*/

	if !bytes.Equal(got, want) {
		t.Errorf("v72 CharacterList wire:\n got %x\nwant %x", got, want)
	}
}

// CharacterViewAllCharacters v72 byte-fixture.
//
// Client read order — CLogin::OnViewAllCharResult @0x5B3F7D, mode 0 (per-world
// character batch, case 0u @0x5b3fb3):
//
//	mode   = Decode1                     // dispatcher mode (0 = char batch)  /*0x5b3fb3*/
//	worldId= Decode1                     // v28                               /*0x5b41d1*/
//	count  = Decode1                     // v13                               /*0x5b41e1*/
//	for each of count entries:
//	    GW_CharacterStat::Decode         // @0x4cf0ee                         /*0x5b4221*/
//	    AvatarLook::Decode               // @0x4d340d                         /*0x5b422f*/
//	    // worldId stored locally (NOT read); NO family byte (viewAll)
//	    rankEnabled = Decode1            // 0 => memset 16; else buffer 16     /*0x5b4243*/
//	        if rankEnabled: DecodeBuffer(16) = 4 rank ints                     /*0x5b425e*/
//
// Byte-identical to the v79 viewAll fixture (viewAll entries already omit the
// family placeholder, so the v73 gate is a no-op here).
//
// packet-audit:verify packet=character/clientbound/CharacterViewAllCharacters version=gms_v72 ida=0x5b3f7d
func TestCharacterViewAllCharactersByteOutputV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	entry := model.NewCharacterListEntry(heroStatsV79(), heroAvatarV79(), true, false, 1, 2, 3, 4)
	got := NewCharacterViewAllCharacters(0, world.Id(1), []model.CharacterListEntry{entry}).Encode(nil, ctx)(nil)

	var want []byte
	want = append(want, 0x00) // mode/code (case 0)    /*0x5b3fb3*/
	want = append(want, 0x01) // worldId               /*0x5b41d1*/
	want = append(want, 0x01) // count = 1             /*0x5b41e1*/
	want = append(want, statBlockV79...)
	want = append(want, avatarBlockV79...)
	want = append(want, 0x01)         // rankEnabled = !gm (no family byte) /*0x5b4243*/
	want = append(want, rank16V79...) // rank ints                         /*0x5b425e*/

	if !bytes.Equal(got, want) {
		t.Errorf("v72 CharacterViewAllCharacters wire:\n got %x\nwant %x", got, want)
	}
}

// CharacterViewAllCount v72 — mode 1 (case 1u @0x5b3fb3):
//	mode  = Decode1                       // dispatcher mode (1)             /*0x5b3fb3*/
//	svrCount = Decode4                    // *((_DWORD*)this+66)             /*0x5b3fd0*/
//	charCount= Decode4                    // *((_DWORD*)this+67)             /*0x5b3fe7*/
//
// packet-audit:verify packet=character/clientbound/CharacterViewAllCount version=gms_v72 ida=0x5b3f7d
func TestCharacterViewAllCountByteOutputV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	got := NewCharacterViewAllCount(1, 2, 3).Encode(nil, ctx)(nil)
	want := []byte{
		0x01,                   // mode/code = 1 (case 1u) /*0x5b3fb3*/
		0x02, 0x00, 0x00, 0x00, // worldCount (Decode4)   /*0x5b3fd0*/
		0x03, 0x00, 0x00, 0x00, // unk (Decode4)          /*0x5b3fe7*/
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v72 CharacterViewAllCount wire: got %x want %x", got, want)
	}
}

// CharacterViewAllSearchFailed v72 — mode 2 (case 2u @0x5b3fb3): after the mode
// byte the client performs NO further wire reads (resets VAC, shows a notice).
//
// packet-audit:verify packet=character/clientbound/CharacterViewAllSearchFailed version=gms_v72 ida=0x5b3f7d
func TestCharacterViewAllSearchFailedByteOutputV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	got := NewCharacterViewAllSearchFailed(2).Encode(nil, ctx)(nil)
	want := []byte{0x02} // mode/code = 2 (case 2u, no further reads) /*0x5b3fb3*/
	if !bytes.Equal(got, want) {
		t.Errorf("v72 CharacterViewAllSearchFailed wire: got %x want %x", got, want)
	}
}

// CharacterViewAllError v72 — default mode (an unhandled mode byte falls through
// to the default branch @0x5b4330 which shows an ERROR modal and performs NO
// further wire reads). Pins against the same base function.
//
// packet-audit:verify packet=character/clientbound/CharacterViewAllError version=gms_v72 ida=0x5b3f7d
func TestCharacterViewAllErrorByteOutputV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	got := NewCharacterViewAllError(8).Encode(nil, ctx)(nil)
	want := []byte{0x08} // mode/code = 8 (default branch, no further reads) /*0x5b4330*/
	if !bytes.Equal(got, want) {
		t.Errorf("v72 CharacterViewAllError wire: got %x want %x", got, want)
	}
}

// AddCharacterEntry v72 byte-fixture.
//
// Client read order — v72 create-character result CLogin::OnDeleteCharacterResult
// @0x5B3C65 (op 14; the IDB symbol is rotated one step off its body — verified by
// dispatch CLogin::OnPacket case 14 @0x5b2516 and by the body):
//
//	code = Decode1                       // result code                     /*0x5b3c80*/
//	// on success (code 0, empty slot found):
//	GW_CharacterStat::Decode             // @0x4cf0ee into empty slot        /*0x5b3d1e*/
//	AvatarLook::Decode                   // @0x4d340d                        /*0x5b3d2c*/
//	// family byte + 16-byte rank buffer are ZEROED locally, NOT read        /*0x5b3d44*/
//
// Matches add_entry.go legacyAddEntry path (GMS v29..v82): [code][stat][avatar]
// with no entry trailer. Byte-identical to the v79 add fixture.
//
// packet-audit:verify packet=character/clientbound/AddCharacterEntry version=gms_v72 ida=0x5b3c65
func TestAddCharacterEntryByteOutputV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	entry := model.NewCharacterListEntry(heroStatsV79(), heroAvatarV79(), false, false, 1, 2, 3, 4)
	got := NewAddCharacterEntry(0, entry).Encode(nil, ctx)(nil)

	var want []byte
	want = append(want, 0x00) // code                 /*0x5b3c80*/
	want = append(want, statBlockV79...)
	want = append(want, avatarBlockV79...)
	// no entry trailer

	if !bytes.Equal(got, want) {
		t.Errorf("v72 AddCharacterEntry wire:\n got %x\nwant %x", got, want)
	}
}

// DeleteCharacterResponse v72 byte-fixture.
//
// Client read order — v72 delete-character result CLogin::OnCheckDuplicatedIDResult
// @0x5B3A18 (op 15; rotated IDB symbol — verified by dispatch case 15 @0x5b2509
// and by the body reading [charId][code]):
//
//	characterId = Decode4                // id of the char to remove        /*0x5b3a3c*/
//	code        = Decode1                // result code                     /*0x5b3a3f*/
//
// Matches DeleteCharacterResponse.Encode exactly ([int charId][byte code]).
//
// packet-audit:verify packet=character/clientbound/DeleteCharacterResponse version=gms_v72 ida=0x5b3a18
func TestDeleteCharacterResponseByteOutputV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	got := NewDeleteCharacterResponse(12345, 0).Encode(nil, ctx)(nil)
	want := []byte{
		0x39, 0x30, 0x00, 0x00, // characterId = 12345 (Decode4) /*0x5b3a3c*/
		0x00, // code (Decode1)                                  /*0x5b3a3f*/
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v72 DeleteCharacterResponse wire: got %x want %x", got, want)
	}
}

// CharacterNameResponse v72 byte-fixture.
//
// Client read order — v72 name-check result CLogin::OnCreateNewCharacterResult
// @0x5B3983 (op 13; rotated IDB symbol — verified by dispatch case 13 @0x5b2523
// and by the body reading [name string][code]):
//
//	name = DecodeStr                     // echoed character name           /*0x5b39a2*/
//	code = Decode1                       // availability/result code        /*0x5b39ad*/
//
// Matches CharacterNameResponse.Encode exactly ([string name][byte code]).
//
// packet-audit:verify packet=character/clientbound/CharacterNameResponse version=gms_v72 ida=0x5b3983
func TestCharacterNameResponseByteOutputV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	got := NewCharacterNameResponse("TestChar", 0).Encode(nil, ctx)(nil)
	want := []byte{
		0x08, 0x00, // name length = 8 (DecodeStr)               /*0x5b39a2*/
		0x54, 0x65, 0x73, 0x74, 0x43, 0x68, 0x61, 0x72, // "TestChar"
		0x00, // code (Decode1)                                  /*0x5b39ad*/
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v72 CharacterNameResponse wire: got %x want %x", got, want)
	}
}

// StatusMessageCashItemExpire v72 byte-fixture — SHOW_STATUS_INFO, op 36.
//
// Client read order — CWvsContext::OnMessage @0x9191EE reads Decode1(nType)
// @0x9191fd then dispatches; the v72 table has arms 0-11 only (SHRUNK from v79's
// 0-13). Case 2 (CASH_ITEM_EXPIRE) → sub_919B59 @0x919b59:
//
//	nType  = Decode1                     // OnMessage dispatch byte (mode)  /*0x9191fd*/
//	itemId = Decode4                     // CItemInfo::GetItemName(itemId)  /*0x919b68*/
//
// The arm reads exactly one Decode4(itemId) then formats StringPool 297
// ("... has expired"). Body has no version gate → byte-identical to v79.
//
// packet-audit:verify packet=character/clientbound/StatusMessageCashItemExpire version=gms_v72 ida=0x919b59
func TestStatusMessageCashItemExpireByteOutputV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	got := NewStatusMessageCashItemExpire(2, 5000000).Encode(nil, ctx)(nil)
	want := []byte{
		0x02,                   // mode = 2 (CASH_ITEM_EXPIRE dispatch byte) /*0x9191fd*/
		0x40, 0x4b, 0x4c, 0x00, // itemId 5000000 (Decode4)                 /*0x919b68*/
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v72 StatusMessageCashItemExpire wire: got %x want %x", got, want)
	}
}

// CharacterAppearanceUpdate v72 byte-fixture — UPDATE_CHAR_LOOK, op 179.
//
// Client read order — CUserRemote::OnAvatarModified @0x88C934 (characterId is
// read by the pool dispatcher before the body):
//
//	flags = Decode1                  // bitfield: &1 look, &2 speed, &4 carry  /*0x88c94d*/
//	if flags & 1:  AvatarLook::Decode  // full avatar/look block (@0x4d340d)    /*0x88c959*/
//	if flags & 2:  Decode1            // riding/vehicle speed (not set here)     /*0x88c9aa*/
//	if flags & 4:  Decode1            // carry-item effect (not set here)        /*0x88c9cc*/
//	coupleMarker   = Decode1         // 0 => none; else 2xDecodeBuffer(8)+Decode4 /*0x88c9de*/
//	friendMarker   = Decode1         // 0 => none; else 2xDecodeBuffer(8)+Decode4 /*0x88ca2b*/
//	marriageMarker = Decode1         // != 0 => 3x Decode4; else zeros          /*0x88ca78*/
//
// As in v79/v83/jms, the marriage if/else has NO trailing unconditional Decode4
// (the else branch @0x88cabd zeroes the slots and reads nothing). Atlas always
// writes flags=1 (look-only) + three ring markers as 0, so the trailing
// WriteInt(0) "completed set item id" is benign trailing slack. Byte-identical
// to the v79 fixture.
//
// packet-audit:verify packet=character/clientbound/CharacterAppearanceUpdate version=gms_v72 ida=0x88c934
func TestCharacterAppearanceUpdateByteOutputV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
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
		0x78, 0x56, 0x34, 0x12, // characterId (WriteInt)                 /*0x88c934 dispatch*/
		0x01,                   // flags = 1 (look-only)                  /*0x88c94d*/
		// --- AvatarLook block (flags & 1) --- @0x4d340d                  /*0x88c959*/
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
		0x00,                   // couple marker (Decode1)                /*0x88c9de*/
		0x00,                   // friend marker (Decode1)                /*0x88ca2b*/
		0x00,                   // marriage marker (Decode1)              /*0x88ca78*/
		0x00, 0x00, 0x00, 0x00, // completed set item id (trailing slack; unread when marriage==0)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v72 CharacterAppearanceUpdate wire:\n got %x\nwant %x", got, want)
	}
}

// CharacterInfo v72 byte-fixture — CHAR_INFO, op 58.
//
// Client read order — CWvsContext::OnCharacterInfo @0x91B961:
//
//	Decode4(charId) /*0x91b996*/, Decode1(level) /*0x91b9bd*/, Decode2(job) /*0x91b9c7*/,
//	Decode2(fame) /*0x91b9d1*/, Decode1(married) /*0x91b9e1*/, DecodeStr(guild) /*0x91b9e8*/,
//	DecodeStr(alliance) /*0x91b9f7*/, Decode1(medalInfo byte) /*0x91ba0c*/,
//	Decode1(first pet flag) /*0x91ba0f*/ → pet loop sub_81697E @0x91ba87 (per pet:
//	  Decode4(templateId), DecodeStr(name), Decode1(level), Decode2(closeness),
//	  Decode1(fullness), Decode2(skill), Decode4(itemId), Decode1(next flag) — bool-term),
//	Decode1(mount flag) /*0x91ba8e*/ + 3×Decode4 /*0x91baa5..*/ → SetTamingMobInfo,
//	Decode1(wish count) /*0x91bad9*/ + count×Decode4 (DecodeBuffer 4*n) /*0x91bb04*/,
//	sub_62F32C @0x91bb37 monster-book (5×Decode4) + medal (Decode4 medalId + Decode2 quest count).
//	NO trailing chair int (the >=87 branch is absent in v72; v72 < 87).
//
// v72 gates == v79 == v83: monster-book present (GMS<=87), chair absent (GMS<87).
// Every version gate in info.go resolves identically for v72 and v79, so the wire
// is byte-identical to the verified v79 golden.
//
// packet-audit:verify packet=character/clientbound/CharacterInfo version=gms_v72 ida=0x91b961
func TestCharacterInfoByteOutputV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	pets := []InfoPet{{Slot: 0, TemplateId: 5000000, Name: "Kitty", Level: 15, Closeness: 200, Fullness: 80}}
	mb := MonsterBookInfo{Level: 5, NormalCards: 10, SpecialCards: 3, TotalCards: 13, Cover: 2380001}
	mount := MountInfo{Active: true, Level: 7, Exp: 1234, Tiredness: 42}
	in := NewCharacterInfo(12345, 50, 100, 10, "TestGuild", pets, []uint32{1002000, 1002001}, 1142007, mb, mount)

	got := in.Encode(nil, ctx)(nil)
	// == v79 golden (info_test.go TestCharacterInfoV79Golden); v72 and v79 share
	// every gate (monster-book present <=87, no chair <87).
	want, _ := hex.DecodeString(
		"393000003264000a00000900546573744775696c6400000001404b4c0005004b697474790fc80050000000000000000107000000d20400002a00000002104a0f00114a0f00050000000a000000030000000d000000e1502400f76c11000000")
	if !bytes.Equal(got, want) {
		t.Errorf("v72 CharacterInfo wire (len got=%d want=%d):\n got %x\nwant %x", len(got), len(want), got, want)
	}
	// Cross-version equality: v72 shape is byte-identical to v83.
	v83 := in.Encode(nil, pt.CreateContext("GMS", 83, 1))(nil)
	if !bytes.Equal(got, v83) {
		t.Errorf("v72 CharacterInfo must equal v83:\n v72 %x\n v83 %x", got, v83)
	}
}

// ---------------------------------------------------------------------------
// Stage E batch 10 — character part C (spawn/damage/expression/chair/sit/keymap
// /effect/upgrade/chalkboard). v72 IDB GMS_v72.1_U_DEVM.exe @13339.
//
// The remote-user clientbound ops route through CUserPool::OnUserRemotePacket
// @0x87c046, which reads Decode4(characterId) @0x87c050 BEFORE dispatch; SPAWN's
// charId is read by CUserPool::OnUserEnterField @0x87bc74 (Decode4 @0x87bc88).
// Every read order below was decompiled on the v72 IDB and matches the verified
// v79 golden (v79_test.go) byte-for-byte, so these fixtures mirror the v79 ones.
// ---------------------------------------------------------------------------

// CharacterSpawn v72 byte-fixture — SPAWN_PLAYER, op 145.
//
// CUserPool::OnUserEnterField @0x87bc74 reads Decode4 characterId (@0x87bc88) then
// builds CUserRemote via sub_888AF4 (the v72 CUserRemote::Init) whose read order is:
//
//	name        = DecodeStr   // this+3884   /*0x888b1f*/ (NO leading level byte)
//	guildName   = DecodeStr   // this+3888   /*0x888b4f*/
//	logoBg      = Decode2     // this+3892   /*0x888b85*/
//	logoBgColor = Decode1     // this+3894   /*0x888b93*/
//	logo        = Decode2     // this+3896   /*0x888ba0*/
//	logoColor   = Decode1     // this+3898   /*0x888bac*/
//	SecondaryStat::DecodeForRemote          // cts (opaque §5) /*0x888bbf*/
//	jobId       = Decode2     // this+11640  /*0x888bd0*/
//	AvatarLook::Decode                      // avatar (opaque §5) /*0x888c09*/
//	choco       = Decode4     // SetCarryItemEffect     /*0x888c1e*/
//	itemEffect  = Decode4     // SetActiveEffectItem     /*0x888c28*/
//	chair       = Decode4     // SetActivePortableChair  /*0x888c32*/
//	x           = Decode2                                /*0x888c42*/
//	y           = Decode2                                /*0x888c4f*/
//	stance      = Decode1     // this+1360   /*0x888c5a*/
//	foothold    = Decode2                                /*0x888c6a*/
//	bShowAdmin  = Decode1                                /*0x888ce7*/
//	pets while(Decode1) loop, 0 terminator              /*0x888d7e*/
//	mountLevel  = Decode4     // this+8788   /*0x888df4*/
//	mountExp    = Decode4     // this+8792   /*0x888e01*/
//	mountTired  = Decode4     // this+8796   /*0x888e0e*/
//	miniRoom    = Decode1 (0 => skip)                    /*0x888e23*/
//	adBoard     = Decode1 (0 => skip)                    /*0x888f89*/
//	couple      = Decode1 (0 => skip)                    /*0x8890b6*/
//	friend      = Decode1 (0 => skip)                    /*0x8890fb*/
//	marriage    = Decode1 (0 => skip)                    /*0x889140*/
//	newYearCard = Decode1 (0 => skip)                    /*0x889185*/
//	effectFlag  = Decode1 (last read)                    /*0x8891c1*/
//
// As in v79 (and unlike v83+): NO leading Decode1 level byte and NO trailing team
// / 2nd-effect byte. Byte-identical to the v79 fixture.
//
// packet-audit:verify packet=character/clientbound/CharacterSpawn version=gms_v72 ida=0x87bc74
func TestCharacterSpawnByteOutputV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	guild := GuildEmblem{Name: "TestGuild", LogoBackground: 1, LogoBackgroundColor: 2, Logo: 3, LogoColor: 4}
	cts := model.NewCharacterTemporaryStat()
	in := NewCharacterSpawn(12345, 50, "TestChar", guild, cts, 100, model.Avatar{}, nil, false, 100, 200, 3, 0)
	got := in.Encode(nil, ctx)(nil)

	// Prefix through the guild emblem — proves NO level byte follows charId.
	var wantPrefix []byte
	wantPrefix = append(wantPrefix, 0x39, 0x30, 0x00, 0x00) // characterId 12345 (Decode4) /*0x87bc88*/
	// NO level byte
	wantPrefix = append(wantPrefix, 0x08, 0x00, 0x54, 0x65, 0x73, 0x74, 0x43, 0x68, 0x61, 0x72) // "TestChar" /*0x888b1f*/
	wantPrefix = append(wantPrefix, 0x09, 0x00, 0x54, 0x65, 0x73, 0x74, 0x47, 0x75, 0x69, 0x6c, 0x64) // "TestGuild" /*0x888b4f*/
	wantPrefix = append(wantPrefix, 0x01, 0x00) // logoBg (Decode2)      /*0x888b85*/
	wantPrefix = append(wantPrefix, 0x02)       // logoBgColor (Decode1) /*0x888b93*/
	wantPrefix = append(wantPrefix, 0x03, 0x00) // logo (Decode2)        /*0x888ba0*/
	wantPrefix = append(wantPrefix, 0x04)       // logoColor (Decode1)   /*0x888bac*/
	if !bytes.HasPrefix(got, wantPrefix) {
		t.Errorf("v72 CharacterSpawn prefix (no level byte):\n got %x\nwant %x", got[:min(len(got), len(wantPrefix))], wantPrefix)
	}

	// Suffix from jobId — proves NO trailing team byte (last wire byte is effectFlag @0x8891c1).
	avatarBytes := model.Avatar{}.Encode(nil, ctx)(nil)
	var wantSuffix []byte
	wantSuffix = append(wantSuffix, 0x64, 0x00) // jobId 100 (Decode2)   /*0x888bd0*/
	wantSuffix = append(wantSuffix, avatarBytes...) // avatar (opaque §5) /*0x888c09*/
	wantSuffix = append(wantSuffix, 0x00, 0x00, 0x00, 0x00) // choco (Decode4)      /*0x888c1e*/
	wantSuffix = append(wantSuffix, 0x00, 0x00, 0x00, 0x00) // itemEffect (Decode4) /*0x888c28*/
	wantSuffix = append(wantSuffix, 0x00, 0x00, 0x00, 0x00) // chair (Decode4)      /*0x888c32*/
	wantSuffix = append(wantSuffix, 0x64, 0x00) // x 100 (Decode2)       /*0x888c42*/
	wantSuffix = append(wantSuffix, 0xc8, 0x00) // y 200 (Decode2)       /*0x888c4f*/
	wantSuffix = append(wantSuffix, 0x03)       // stance (Decode1)      /*0x888c5a*/
	wantSuffix = append(wantSuffix, 0x00, 0x00) // foothold (Decode2)    /*0x888c6a*/
	wantSuffix = append(wantSuffix, 0x00)       // bShowAdmin (Decode1)  /*0x888ce7*/
	wantSuffix = append(wantSuffix, 0x00)       // pets terminator       /*0x888d7e*/
	wantSuffix = append(wantSuffix, 0x01, 0x00, 0x00, 0x00) // mountLevel (Decode4) /*0x888df4*/
	wantSuffix = append(wantSuffix, 0x00, 0x00, 0x00, 0x00) // mountExp (Decode4)   /*0x888e01*/
	wantSuffix = append(wantSuffix, 0x00, 0x00, 0x00, 0x00) // mountTired (Decode4) /*0x888e0e*/
	wantSuffix = append(wantSuffix, 0x00) // miniRoom (Decode1)    /*0x888e23*/
	wantSuffix = append(wantSuffix, 0x00) // adBoard (Decode1)     /*0x888f89*/
	wantSuffix = append(wantSuffix, 0x00) // couple (Decode1)      /*0x8890b6*/
	wantSuffix = append(wantSuffix, 0x00) // friend (Decode1)      /*0x8890fb*/
	wantSuffix = append(wantSuffix, 0x00) // marriage (Decode1)    /*0x889140*/
	wantSuffix = append(wantSuffix, 0x00) // newYearCard (Decode1) /*0x889185*/
	wantSuffix = append(wantSuffix, 0x00) // effectFlag (Decode1, last read) /*0x8891c1*/
	if !bytes.HasSuffix(got, wantSuffix) {
		n := len(wantSuffix)
		if n > len(got) {
			n = len(got)
		}
		t.Errorf("v72 CharacterSpawn suffix (no team byte):\n got %x\nwant %x", got[len(got)-n:], wantSuffix)
	}
}

// CharacterDamage v72 byte-fixture — DAMAGE_PLAYER, op 174.
//
// CUserRemote::OnHit @0x88c5ad read order (physical, attackIdx = -1; -1 > -2 so the
// LABEL_32 shortcut is NOT taken):
//
//	attackIdx        = Decode1   /*0x88c5ca*/
//	damage           = Decode4   /*0x88c5e0*/
//	monsterTemplate  = Decode4   /*0x88c5f4*/
//	left             = Decode1   /*0x88c602*/
//	stance           = Decode1 (0 => inner block skipped) /*0x88c756*/
//	stanceRelated    = Decode1   /*0x88c870*/
//	damage (repeat)  = Decode4   /*0x88c897*/
//
// characterId(4) is read by the pool dispatcher (Decode4 @0x87c050). No bGuard byte
// (GMS>=95 only; v72 < 95). Byte-identical to the v79 fixture.
//
// packet-audit:verify packet=character/clientbound/CharacterDamage version=gms_v72 ida=0x88c5ad
func TestCharacterDamageByteOutputV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	got := NewCharacterDamage(1234, model.DamageTypePhysical, 500, 100100, true).Encode(nil, ctx)(nil)
	want := []byte{
		0xd2, 0x04, 0x00, 0x00, // characterId 1234 (dispatcher Decode4) /*0x87c050*/
		0xff,                   // attackIdx -1 (Decode1)                /*0x88c5ca*/
		0xf4, 0x01, 0x00, 0x00, // damage 500 (Decode4)                  /*0x88c5e0*/
		0x04, 0x87, 0x01, 0x00, // monsterTemplateId 100100 (Decode4)    /*0x88c5f4*/
		0x01,                   // left (Decode1)                        /*0x88c602*/
		0x00,                   // stance (Decode1)                      /*0x88c756*/
		0x00,                   // stanceRelated (Decode1)               /*0x88c870*/
		0xf4, 0x01, 0x00, 0x00, // damage repeated (Decode4)             /*0x88c897*/
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v72 CharacterDamage wire:\n got %x\nwant %x", got, want)
	}
}

// CharacterExpression v72 byte-fixture — FACIAL_EXPRESSION, op 175.
//
// Read inline in CUserPool::OnUserRemotePacket case 175 @0x87c0c6: v6 = Decode4,
// then CAvatar::SetEmotion(RemoteUser+124, v6, -1) — NO duration, NO byItemOption
// (GMS>87 / JMS only; v72 < 87). characterId(4) is read by the dispatcher
// (Decode4 @0x87c050). Byte-identical to the v79 fixture.
//
// packet-audit:verify packet=character/clientbound/CharacterExpression version=gms_v72 ida=0x44cb1a
func TestCharacterExpressionByteOutputV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	got := NewCharacterExpression(12345, 5, 3000).Encode(nil, ctx)(nil)
	want := []byte{
		0x39, 0x30, 0x00, 0x00, // characterId 12345 (dispatcher Decode4) /*0x87c050*/
		0x05, 0x00, 0x00, 0x00, // expression 5 (Decode4)                 /*0x87c0c6*/
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v72 CharacterExpression wire: got %x want %x", got, want)
	}
}

// CharacterChairShow v72 byte-fixture — SHOW_CHAIR, op 178.
//
// Read inline in CUserPool::OnUserRemotePacket case 178 @0x87c12e:
// *((_DWORD*)RemoteUser + 2902) = Decode4(chairId); characterId(4) is read by the
// dispatcher (Decode4 @0x87c050). Two LE uint32s. Byte-identical to the v79 fixture.
//
// packet-audit:verify packet=character/clientbound/CharacterChairShow version=gms_v72 ida=0x87c046
func TestCharacterChairShowByteOutputV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	got := NewCharacterChairShow(1234, 3010000).Encode(nil, ctx)(nil)
	want := []byte{
		0xd2, 0x04, 0x00, 0x00, // characterId 1234 (dispatcher Decode4) /*0x87c050*/
		0xd0, 0xed, 0x2d, 0x00, // chairId 3010000 (Decode4)             /*0x87c12e*/
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v72 CharacterChairShow wire: got %x want %x", got, want)
	}
}

// CharacterSitResult v72 byte-fixture — CANCEL_CHAIR clientbound, op 187.
//
// CUserLocal::OnSitResult @0x865e68: flag = Decode1; if flag != 0 then
// nSeat = Decode2 @0x865ee5 (else stand-up branch reads nothing more). Byte-identical
// to the v79 fixture.
//
// packet-audit:verify packet=character/clientbound/CharacterSitResult version=gms_v72 ida=0x865e68
func TestCharacterSitResultByteOutputV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	// sit: flag(1)=1 then chairId(2, LE) read.
	gotSit := NewCharacterSit(17).Encode(nil, ctx)(nil)
	wantSit := []byte{0x01, 0x11, 0x00} // flag=1, chairId 17 (Decode2) /*0x865ee5*/
	if !bytes.Equal(gotSit, wantSit) {
		t.Errorf("v72 CharacterSitResult sit: got %x want %x", gotSit, wantSit)
	}
	// cancel: flag(1)=0, stand-up branch reads nothing else.
	gotCancel := NewCharacterCancelSit().Encode(nil, ctx)(nil)
	wantCancel := []byte{0x00} // flag=0
	if !bytes.Equal(gotCancel, wantCancel) {
		t.Errorf("v72 CharacterSitResult cancel: got %x want %x", gotCancel, wantCancel)
	}
}

// CharacterKeyMap v72 byte-fixture — KEYMAP, op 298.
//
// CFuncKeyMappedMan::OnInit @0x551370: reset = Decode1 @0x551378; if reset == 0 the
// client loops v5 = 89 FUNCKEY_MAPPED::Decode (each DecodeBuffer(5) = nType[1]+nID[4])
// @0x5513ad, then memcpy 0x1BD = 445 = 89*5 @0x5513d5. v72 reads 89 entries (== v79,
// NOT the 90 the v83 codec historically emits); keymap.go gates the count to 89 for
// GMS < 83.
//
// packet-audit:verify packet=character/clientbound/CharacterKeyMap version=gms_v72 ida=0x551370
func TestCharacterKeyMapByteOutputV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	keys := map[int32]KeyBinding{
		2:  {KeyType: 4, KeyAction: 10},
		16: {KeyType: 4, KeyAction: 8},
		41: {KeyType: 4, KeyAction: 11},
	}
	got := NewCharacterKeyMap(keys).Encode(nil, ctx)(nil)

	var want []byte
	want = append(want, 0x00) // not-reset flag (Decode1 == 0) /*0x551378*/
	for i := int32(0); i < 89; i++ { // v72 reads 89 FUNCKEY_MAPPED entries /*0x5513ad v5=89*/
		if k, ok := keys[i]; ok {
			want = append(want, byte(k.KeyType)) // nType /*0x5513b4*/
			want = append(want, byte(k.KeyAction), byte(k.KeyAction>>8), byte(k.KeyAction>>16), byte(k.KeyAction>>24))
		} else {
			want = append(want, 0x00, 0x00, 0x00, 0x00, 0x00)
		}
	}
	if len(got) != 1+89*5 {
		t.Fatalf("v72 CharacterKeyMap length: got %d, want %d", len(got), 1+89*5)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v72 CharacterKeyMap wire:\n got %x\nwant %x", got, want)
	}

	gotReset := NewCharacterKeyMapResetToDefault().Encode(nil, ctx)(nil)
	if !bytes.Equal(gotReset, []byte{0x01}) {
		t.Errorf("v72 CharacterKeyMap reset: got %x want 01", gotReset)
	}
}

// EffectQuest v72 byte-fixture — SHOW_FOREIGN_EFFECT (180) / SHOW_ITEM_GAIN_INCHAT (188).
//
// CUser::OnEffect @0x846e1e dispatches on Decode1(mode); mode 3 (case 3u) reads:
//
//	count = Decode1                        /*0x8471dd*/
//	if count == 0:
//	    message = DecodeStr                /*0x847357*/
//	    nEffect = Decode4                  /*0x847397*/
//	else (count>0): repeat count times:
//	    itemId  = Decode4                  /*0x8471f9*/
//	    amount  = Decode4                  /*0x84720a*/
//
// EffectQuestForeign prepends Decode4(characterId) consumed by
// CUserPool::OnUserRemotePacket before dispatch. Byte-identical to the v79 fixture.
//
// packet-audit:verify packet=character/clientbound/EffectQuest version=gms_v72 ida=0x846e1e
func TestEffectQuestByteOutputV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)

	t.Run("message", func(t *testing.T) {
		got := NewEffectQuest(3, "Hello", 0x10, nil).Encode(nil, ctx)(nil)
		want := []byte{
			0x03,                         // mode (OnEffect selector = case 3) /*0x846e31*/
			0x00,                         // count = 0 (Decode1)               /*0x8471dd*/
			0x05, 0x00, 0x48, 0x65, 0x6c, 0x6c, 0x6f, // "Hello" (DecodeStr)    /*0x847357*/
			0x10, 0x00, 0x00, 0x00, // nEffect 0x10 (Decode4)                  /*0x847397*/
		}
		if !bytes.Equal(got, want) {
			t.Errorf("v72 EffectQuest message:\n got %x\nwant %x", got, want)
		}
	})

	t.Run("rewards", func(t *testing.T) {
		got := NewEffectQuest(3, "", 0, []QuestReward{{ItemId: 2000000, Amount: 5}}).Encode(nil, ctx)(nil)
		want := []byte{
			0x03,                   // mode                                  /*0x846e31*/
			0x01,                   // count = 1 (Decode1)                   /*0x8471dd*/
			0x80, 0x84, 0x1e, 0x00, // itemId 2000000 (Decode4)              /*0x8471f9*/
			0x05, 0x00, 0x00, 0x00, // amount 5 (Decode4)                    /*0x84720a*/
		}
		if !bytes.Equal(got, want) {
			t.Errorf("v72 EffectQuest rewards:\n got %x\nwant %x", got, want)
		}
	})

	t.Run("foreign", func(t *testing.T) {
		got := NewEffectQuestForeign(0x12345678, 3, "Hi", 7, nil).Encode(nil, ctx)(nil)
		want := []byte{
			0x78, 0x56, 0x34, 0x12, // characterId (dispatcher Decode4)      /*0x87c050*/
			0x03,             // mode                                        /*0x846e31*/
			0x00,             // count = 0                                   /*0x8471dd*/
			0x02, 0x00, 0x48, 0x69, // "Hi"                                  /*0x847357*/
			0x07, 0x00, 0x00, 0x00, // nEffect 7 (Decode4)                   /*0x847397*/
		}
		if !bytes.Equal(got, want) {
			t.Errorf("v72 EffectQuestForeign:\n got %x\nwant %x", got, want)
		}
	})
}

// ItemUpgrade v72 byte-fixture — SHOW_SCROLL_EFFECT, op 152.
//
// CUser::ShowItemUpgradeEffect @0x84315c reads, after the dispatcher's Decode4
// characterId, exactly four Decode1 flags:
//
//	success         = Decode1  /*0x843191*/
//	cursed          = Decode1  /*0x84319b*/
//	legendarySpirit = Decode1  /*0x8431ac*/
//	whiteScroll     = Decode1  /*0x8431b7*/
//
// v72 (GMS < 87) reads NO Decode4 enchantCategory and NO 5th Decode1 enchantResultFlag
// (both are GMS>87 / JMS additions). Matches item_upgrade.go's <=87 path; identical to
// the v83/v87 wire.
//
// packet-audit:verify packet=character/clientbound/ItemUpgrade version=gms_v72 ida=0x84315c
func TestItemUpgradeByteOutputV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	got := NewItemUpgrade(12345, true, false, true, false).Encode(nil, ctx)(nil)
	want := []byte{
		0x39, 0x30, 0x00, 0x00, // characterId 12345 (dispatcher Decode4)
		0x01, // success = true (Decode1)         /*0x843191*/
		0x00, // cursed = false (Decode1)         /*0x84319b*/
		0x01, // legendarySpirit = true (Decode1) /*0x8431ac*/
		0x00, // whiteScroll = false (Decode1)    /*0x8431b7*/
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v72 ItemUpgrade wire:\n got %x\nwant %x", got, want)
	}
}

// ChalkboardUse v72 byte-fixture — CHALKBOARD, op 149.
//
// CUser::OnADBoard @0x846c4c reads, after the dispatcher's Decode4 characterId:
//
//	active  = Decode1  // if-guard (0 => clear, no message)
//	message = DecodeStr // only when active                 /*0x846cae*/
//
// Byte-identical to the v79 fixture.
//
// packet-audit:verify packet=character/clientbound/ChalkboardUse version=gms_v72 ida=0x846c4c
func TestChalkboardUseByteOutputV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)

	t.Run("active", func(t *testing.T) {
		got := NewChalkboardUse(1234, "Hi").Encode(nil, ctx)(nil)
		want := []byte{
			0xD2, 0x04, 0x00, 0x00, // characterId (dispatcher prefix)
			0x01,       // active = 1 (Decode1)
			0x02, 0x00, // message len = 2 (DecodeStr) /*0x846cae*/
			0x48, 0x69, // "Hi"
		}
		if !bytes.Equal(got, want) {
			t.Errorf("v72 ChalkboardUse active:\n got %x\nwant %x", got, want)
		}
	})

	t.Run("clear", func(t *testing.T) {
		got := NewChalkboardClear(1234).Encode(nil, ctx)(nil)
		want := []byte{
			0xD2, 0x04, 0x00, 0x00, // characterId (dispatcher prefix)
			0x00, // active = 0 (Decode1 false)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("v72 ChalkboardUse clear:\n got %x\nwant %x", got, want)
		}
	})
}

// StatusMessageDropPickUpMeso v72 byte-fixture — SHOW_STATUS_INFO, op 36.
//
// Client read order — CWvsContext::OnMessage @0x9191EE case 0 → drops arm
// sub_9192D0 @0x9192d0. Inner drop-type Decode1@0x9192ef == 1 (meso) branch:
//
//	mode      = Decode1  // OnMessage dispatch byte @0x9191fd
//	dropType  = Decode1  // inner type = 1 (meso)   @0x9192ef
//	meso      = Decode4  // @0x91930b
//	cafeBonus = Decode2  // @0x919314
//
// LEGACY DIVERGENCE vs v79: v72 reads NO partial-pickup flag between dropType and
// meso (confirmed by disassembly — the meso branch has a single leading Decode1
// then Decode4+Decode2). v79 sub_96AEEC meso branch reads Decode1(partial)@0x96af28
// first. status_message.go gates the partial byte on GMS>=79; the v72 wire omits it.
//
// packet-audit:verify packet=character/clientbound/StatusMessageDropPickUpMeso version=gms_v72 ida=0x9192d0
func TestStatusMessageDropPickUpMesoByteOutputV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	got := NewStatusMessageDropPickUpMeso(0, true, 1000, 0).Encode(nil, ctx)(nil)
	want := []byte{
		0x00,                   // mode = 0 (drops dispatch byte)   /*0x9191fd*/
		0x01,                   // inner drop-type = 1 (meso)       /*0x9192ef*/
		// NO partial-pickup byte (v72 < 79)
		0xe8, 0x03, 0x00, 0x00, // meso 1000 (Decode4)              /*0x91930b*/
		0x00, 0x00, // internetCafeBonus 0 (Decode2)   /*0x919314*/
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v72 StatusMessageDropPickUpMeso wire: got %x want %x", got, want)
	}
}

// StatusMessageIncreaseExperience v72 byte-fixture — SHOW_STATUS_INFO, op 36.
//
// Client read order — CWvsContext::OnMessage @0x9191EE case 3 → sub_919E04
// @0x919e04 (all Decode1 unless noted, verified by disassembly — 8 Decode1 +
// 4 Decode4):
//
//	mode                 = Decode1  // dispatch byte @0x9191fd
//	white                = Decode1  // @0x919e1c
//	amount               = Decode4  // @0x919e29
//	inChat               = Decode1  // @0x919e32
//	monsterBookBonus     = Decode4  // @0x919e3f
//	mobEventBonusPct     = Decode1  // @0x919e49
//	partyBonusPct        = Decode1  // @0x919e56
//	weddingBonusEXP      = Decode4  // @0x919e63
//	[mobEventBonusPct>0] playTimeHour        = Decode1  // @0x919e74
//	[inChat] questBonusRate                  = Decode1  // @0x919e8b
//	    [questBonusRate>0] questBonusRemain   = Decode1  // @0x919ea2
//	partyBonusEventRate  = Decode1  // @0x919eb4
//	partyBonusExp        = Decode4  // @0x919ec1 (the ONLY trailing int)
//
// LEGACY DIVERGENCE vs v79: v72 reads exactly ONE trailing Decode4 after
// partyBonusEventRate; v79 sub_96BD0D reads THREE (partyBonusExp @0x96bdcb,
// itemBonusEXP @0x96bdd5, premiumIPExp @0x96bde0). Everything up to and including
// partyBonusEventRate is byte-identical. status_message.go gates the 2nd/3rd
// trailing ints (and rainbow/ring/cake) on GMS>=79; the v72 wire stops at 1.
//
// packet-audit:verify packet=character/clientbound/StatusMessageIncreaseExperience version=gms_v72 ida=0x919e04
func TestStatusMessageIncreaseExperienceByteOutputV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	got := NewStatusMessageIncreaseExperience(3, true, 500, true, 10, 5, 0, 0, 2, 3, 1, 0, 0, 50, 0, 0, 100, 200).Encode(nil, ctx)(nil)
	want := []byte{
		0x03,                   // mode = 3 (IncEXP dispatch byte)       /*0x9191fd*/
		0x01,                   // white = true (Decode1)               /*0x919e1c*/
		0xf4, 0x01, 0x00, 0x00, // amount 500 (Decode4)                 /*0x919e29*/
		0x01,                   // inChat = true (Decode1)              /*0x919e32*/
		0x0a, 0x00, 0x00, 0x00, // monsterBookBonus 10 (Decode4)        /*0x919e3f*/
		0x05,                   // mobEventBonusPct 5 (Decode1)         /*0x919e49*/
		0x00,                   // partyBonusPct 0 (Decode1)            /*0x919e56*/
		0x00, 0x00, 0x00, 0x00, // weddingBonusEXP 0 (Decode4)          /*0x919e63*/
		0x02,                   // playTimeHour 2 (Decode1, mob>0)      /*0x919e74*/
		0x03,                   // questBonusRate 3 (Decode1, inChat)   /*0x919e8b*/
		0x01,                   // questBonusRemain 1 (Decode1, rate>0) /*0x919ea2*/
		0x00,                   // partyBonusEventRate 0 (Decode1)      /*0x919eb4*/
		0x00, 0x00, 0x00, 0x00, // partyBonusExp 0 (Decode4, sole trailing int) /*0x919ec1*/
		// NO itemBonusEXP / premiumIPExp (v72 < 79)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v72 StatusMessageIncreaseExperience wire:\n got %x\nwant %x", got, want)
	}
}
