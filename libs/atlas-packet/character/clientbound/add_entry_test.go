package clientbound

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=character/clientbound/AddCharacterEntry version=gms_v83 ida=0x5fa26c
// packet-audit:verify packet=character/clientbound/AddCharacterEntry version=gms_v84 ida=0x60f268
// packet-audit:verify packet=character/clientbound/AddCharacterEntry version=gms_v87 ida=0x631b13
// packet-audit:verify packet=character/clientbound/AddCharacterEntry version=gms_v95 ida=0x5dab90
// packet-audit:verify packet=character/clientbound/AddCharacterEntry version=jms_v185 ida=0x66ffa8
func TestAddCharacterEntryRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)

			stats := model.NewCharacterStatistics(
				12345, "TestChar", 1, 3, 20001, 30001,
				[3]uint64{100, 200, 300},
				50, 111,
				40, 30, 20, 10,
				5000, 5000, 3000, 3000,
				5, false, 3,
				123456, 100, 5000,
				100000, 2,
			)
			avatar := model.NewAvatar(1, 3, 20001, false, 30001, nil, nil, nil)
			entry := model.NewCharacterListEntry(stats, avatar, false, false, 10, 1, 5, 2)

			input := NewAddCharacterEntry(0, entry)
			output := AddCharacterEntry{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)

			if output.Code() != input.Code() {
				t.Errorf("code: got %v, want %v", output.Code(), input.Code())
			}
			if output.Character().Statistics().Id() != input.Character().Statistics().Id() {
				t.Errorf("characterId: got %v, want %v", output.Character().Statistics().Id(), input.Character().Statistics().Id())
			}
			if output.Character().Statistics().Name() != input.Character().Statistics().Name() {
				t.Errorf("name: got %v, want %v", output.Character().Statistics().Name(), input.Character().Statistics().Name())
			}
			// GMS v28 Avatar.Encode skips gender/skin/face/hair (written only by CharacterStatistics in that version).
			if !(v.Region == "GMS" && v.MajorVersion <= 28) {
				if output.Character().Avatar().Gender() != input.Character().Avatar().Gender() {
					t.Errorf("gender: got %v, want %v", output.Character().Avatar().Gender(), input.Character().Avatar().Gender())
				}
			}
		})
	}
}

// AddCharacterEntry v48 byte-fixture (GMS_v48_1_DEVM.exe, port 13337).
//
// Client read order — CLogin::OnCreateNewCharacterResult sub_501973 @0x501973:
//
//	Decode1(code) /*0x501987*/ → on success GW_CharacterStat::Decode (sub_49B627)
//	→ AvatarLook::Decode (sub_49E1E0) into a free slot, then the family byte and
//	16-byte rank buffer are zeroed LOCALLY (not read from the wire). So the legacy
//	v29..v82 wire is [code][GW_CharacterStat][AvatarLook] with NO list-entry trailer
//	(legacyAddEntry gate in add_entry.go). GW_CharacterStat / AvatarLook use the v48
//	single-pet legacy shape (single 8-byte pet in stat, single 4-byte pet in avatar);
//	see TestCharacterListByteOutputV48 for the byte-by-byte field trace.
//
// packet-audit:verify packet=character/clientbound/AddCharacterEntry version=gms_v48 ida=0x501973
func TestAddCharacterEntryByteOutputV48(t *testing.T) {
	ctx := pt.CreateContext("GMS", 48, 1)

	stats := model.NewCharacterStatistics(
		0x01020304, "Hero", 0, 0, 0x4D2, 0x7B, [3]uint64{0, 0, 0},
		0x0A, 0x64, 4, 5, 6, 7, 0x64, 0x64, 0x32, 0x32, 3, false, 2, 0, 8, 0, 0x0BB8, 0,
	)
	avatar := model.NewAvatar(0, 0, 0x4D2, false, 0x7B, nil, nil, nil)
	entry := model.NewCharacterListEntry(stats, avatar, false, false, 1, 2, 3, 4)

	got := NewAddCharacterEntry(0, entry).Encode(nil, ctx)(nil)

	want := []byte{
		0x00, // code (Decode1)                                    /*0x501987*/

		// --- GW_CharacterStat block --- sub_49B627 @0x49b627
		0x04, 0x03, 0x02, 0x01, // id
		0x48, 0x65, 0x72, 0x6f, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // "Hero"+pad13
		0x00,                   // gender
		0x00,                   // skin
		0xd2, 0x04, 0x00, 0x00, // face
		0x7b, 0x00, 0x00, 0x00, // hair
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // SINGLE pet long (8 bytes)
		0x0a,       // level
		0x64, 0x00, // jobId
		0x04, 0x00, // str
		0x05, 0x00, // dex
		0x06, 0x00, // int
		0x07, 0x00, // luck
		0x64, 0x00, // hp
		0x64, 0x00, // maxHp
		0x32, 0x00, // mp
		0x32, 0x00, // maxMp
		0x03, 0x00, // ap
		0x02, 0x00, // sp
		0x00, 0x00, 0x00, 0x00, // exp
		0x08, 0x00, // fame
		0xb8, 0x0b, 0x00, 0x00, // mapId
		0x00, // spawnPoint

		// --- AvatarLook block --- sub_49E1E0 @0x49e1e0
		0x00,                   // gender
		0x00,                   // skin
		0xd2, 0x04, 0x00, 0x00, // face
		0x01,                   // !mega
		0x7b, 0x00, 0x00, 0x00, // hair
		0xff,                   // equip terminator
		0xff,                   // masked terminator
		0x00, 0x00, 0x00, 0x00, // cash weapon
		0x00, 0x00, 0x00, 0x00, // SINGLE pet int (4 bytes)

		// --- NO entry trailer: family/rank zeroed locally (legacy add) ---
	}
	if !bytes.Equal(got, want) {
		t.Errorf("AddCharacterEntry v48 bytes:\n got %x\nwant %x", got, want)
	}
}

// TestAddCharacterEntryJMSGolden pins the full jms_v185 wire for a ranked (non-GM)
// AddCharacterEntry. jms read order is CLogin::OnCreateNewCharacterResult @0x66ffa8:
//
//	Decode1(code) → GW_CharacterStat::Decode @0x50ec17 → AvatarLook::Decode @0x51517e,
//
// then the list-entry trailer (rankEnabled byte + 4 rank ints; viewAll=false adds a
// leading 0 byte). The jms GW_CharacterStat block is 18 bytes wider than v83's (extra
// stat fields), which is exactly the jms version delta and why the body is 159 bytes
// (v83 is 141). Bytes hand-derived from the codec, which 8b confirmed emits jms-correct
// GW_CharacterStat / AvatarLook wire.
func TestAddCharacterEntryJMSGolden(t *testing.T) {
	v := pt.Variants[4] // JMS v185
	ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)

	stats := model.NewCharacterStatistics(
		12345, "TestChar", 1, 3, 20001, 30001,
		[3]uint64{100, 200, 300},
		50, 111,
		40, 30, 20, 10,
		5000, 5000, 3000, 3000,
		5, false, 3,
		123456, 100, 5000,
		100000, 2,
	)
	avatar := model.NewAvatar(1, 3, 20001, false, 30001, nil, nil, nil)
	entry := model.NewCharacterListEntry(stats, avatar, false, false, 10, 1, 5, 2)

	got := NewAddCharacterEntry(0, entry).Encode(nil, ctx)(nil)
	want, _ := hex.DecodeString(
		"0039300000546573744368617200000000000103214e0000317500006400000000000000c8000000000000002c01000000000000326f0028001e0014000a0088138813b80bb80b0500030040e20100640088130000a086010002000000000000000000000000000000000000000000000103214e00000131750000ffff0000000000000000000000000000000000010a000000010000000500000002000000")
	if !bytes.Equal(got, want) {
		t.Errorf("jms AddCharacterEntry wire (len got=%d want=%d):\n got %x\nwant %x", len(got), len(want), got, want)
	}
}

// AddCharacterError v48 — ADD_NEW_CHAR_ENTRY error path (op 14). The v48 create
// handler CLogin::OnCreateNewCharacterResult sub_501973 @0x501973 reads
// Decode1(code) @0x50198e; a non-zero code (or no free slot) branches to an error
// dialog sub_50FF3B(18) @0x501ad1 with NO stat/avatar body read from the wire.
// AddCharacterError.Encode writes the single [code] byte. == v61.
//
// packet-audit:verify packet=character/clientbound/AddCharacterError version=gms_v48 ida=0x501973
func TestAddCharacterErrorByteOutputV48(t *testing.T) {
	ctx := pt.CreateContext("GMS", 48, 1)
	got := NewAddCharacterError(9).Encode(nil, ctx)(nil)
	want := []byte{0x09} // code (Decode1) /*0x50198e*/
	if !bytes.Equal(got, want) {
		t.Errorf("v48 AddCharacterError wire: got %x want %x", got, want)
	}
}
