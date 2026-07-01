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
