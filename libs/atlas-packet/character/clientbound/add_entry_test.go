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

// TestAddCharacterEntryJMSGolden pins the full jms_v185 wire for a ranked (non-GM)
// AddCharacterEntry. jms read order is CLogin::OnCreateNewCharacterResult @0x66ffa8:
//   Decode1(code) → GW_CharacterStat::Decode @0x50ec17 → AvatarLook::Decode @0x51517e,
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
