package clientbound

import (
	"bytes"
	"encoding/binary"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-packet/interaction"
	"github.com/Chronicle20/atlas/libs/atlas-packet/model"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// singleEquipAvatar has exactly one equip slot so its Encode output is
// deterministic (model.Avatar backs equips with a map; multi-entry iteration
// order is random). A stable avatar blob is required for an exact byte fixture.
func singleEquipAvatar() model.Avatar {
	equip := map[slot.Position]uint32{5: 1040002}
	return model.NewAvatar(0, 1, 20000, false, 30000, equip, map[slot.Position]uint32{}, map[int8]uint32{})
}

// le16 / le32 encode a little-endian short / int, matching response.Writer.
func le16(v uint16) []byte {
	b := make([]byte, 2)
	binary.LittleEndian.PutUint16(b, v)
	return b
}

func le32(v uint32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, v)
	return b
}

// ascii mirrors response.Writer.WriteAsciiString for pure-ASCII input:
// uint16 LE length + raw bytes.
func ascii(s string) []byte {
	return append(le16(uint16(len(s))), []byte(s)...)
}

// Game room-enter blob (task-7b). Verified byte-for-byte against the decompiled
// client read order on gms_v83 (OnEnterResultBase 0x65ec3d + COmokDlg::OnEnterResult
// 0x6e388e; fixture TestInteractionMiniGameRoomBytes) and gms_v95
// (OnEnterResultBase 0x638e30 + COmokDlg::OnEnterResult 0x680e70; fixture
// TestInteractionMiniGameRoomBytesV95, which additionally asserts the v84+/JMS
// per-avatar uint16 jobCode). The vtable+92 IsEntrusted() predicate is 0 for both
// game dialogs (sub_48315F `return 0`), so the owner-slot-0 int32 branch is dead —
// every occupant is a full avatar (ida-notes.md §G5 "Room-enter blob — FULL RESOLUTION").
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionMiniGameRoom version=gms_v83 ida=0x65ec3d
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionMiniGameRoom version=gms_v95 ida=0x638e30
//
// TestInteractionMiniGameRoomBytes is an encode-only byte fixture asserting the
// EXACT gms_v83 wire sequence of the Omok / Match Cards room-enter blob against
// the decompiled client read order (ida-notes.md §G5 "Room-enter blob — FULL
// RESOLUTION"):
//
//	mode, roomType, capacity, yourSlot,
//	AVATAR list  (0xFF-terminated): {slot, AvatarLook blob, name}...
//	RECORD list  (0xFF-terminated): {slot, 5×int32}...
//	title, gameKind, tournament [, round]
//
// The two lists are SEPARATE (avatars first, then records) — NOT interleaved —
// and every occupant including owner slot 0 is a full avatar (the vtable+92
// IsEntrusted() int32 branch is dead for games; §G5). This is the correction
// the design (§6.1) called for over the interleaved single-list model.
//
// The per-avatar uint16 jobCode after each name is version-gated: v83 does NOT
// read it (0x65ec3d, disasm-verified); v84 (0x674aa6), v87 (0x698f32),
// v95 (0x638e30 `m_anJobCode[i] = Decode2()`) and jms v185 (0x6dabdb) DO.
func miniGameRoomWant(avatarBytes []byte, withJobCode bool) []byte {
	var want []byte
	want = append(want, 5) // mode (ROOM / EnterResult)
	want = append(want, 1) // roomType = Omok
	want = append(want, 2) // capacity (m_nMaxUsers)
	want = append(want, 0) // yourSlot (m_nMyPosition)
	// avatar list
	want = append(want, 0)                 // slot 0
	want = append(want, avatarBytes...)    // AvatarLook blob
	want = append(want, ascii("Owner")...) // name
	if withJobCode {
		want = append(want, le16(412)...) // jobCode (m_anJobCode[0])
	}
	want = append(want, 1)                 // slot 1
	want = append(want, avatarBytes...)    // AvatarLook blob
	want = append(want, ascii("Guest")...) // name
	if withJobCode {
		want = append(want, le16(230)...) // jobCode (m_anJobCode[1])
	}
	want = append(want, 0xFF) // end avatar list
	// record list (separate)
	want = append(want, 0)                   // slot 0
	want = append(want, le32(1)...)          // Unknown
	want = append(want, le32(10)...)         // Wins
	want = append(want, le32(2)...)          // Ties
	want = append(want, le32(3)...)          // Losses
	want = append(want, le32(500)...)        // Points
	want = append(want, 1)                   // slot 1
	want = append(want, le32(1)...)          // Unknown
	want = append(want, le32(4)...)          // Wins
	want = append(want, le32(1)...)          // Ties
	want = append(want, le32(6)...)          // Losses
	want = append(want, le32(250)...)        // Points
	want = append(want, 0xFF)                // end record list
	want = append(want, ascii("FunRoom")...) // title
	want = append(want, 0)                   // gameKind
	want = append(want, 0)                   // tournament = false
	return want
}

func miniGameRoomFixtureInput() InteractionMiniGameRoom {
	avatar := singleEquipAvatar()
	players := []MiniGameRoomPlayer{
		{Slot: 0, Avatar: avatar, Name: "Owner", JobCode: 412, Record: interaction.GameRecord{Unknown: 1, Wins: 10, Ties: 2, Losses: 3, Points: 500}},
		{Slot: 1, Avatar: avatar, Name: "Guest", JobCode: 230, Record: interaction.GameRecord{Unknown: 1, Wins: 4, Ties: 1, Losses: 6, Points: 250}},
	}
	return NewInteractionMiniGameRoom(5, interaction.OmokRoomType, 2, 0, players, "FunRoom", 0, false, 0)
}

func TestInteractionMiniGameRoomBytes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 83, 1)
	avatarBytes := singleEquipAvatar().Encode(l, ctx)(nil)

	got := test.Encode(t, ctx, miniGameRoomFixtureInput().Encode, nil)
	want := miniGameRoomWant(avatarBytes, false) // v83: NO per-avatar jobCode

	if !bytes.Equal(got, want) {
		t.Fatalf("byte mismatch:\n got  %x\n want %x", got, want)
	}
}

// TestInteractionMiniGameRoomBytesV95 asserts the EXACT gms_v95 wire sequence:
// identical to v83 except each avatar-list entry carries a trailing uint16
// jobCode (v95 OnEnterResultBase @0x638e30: `m_anJobCode[i] = Decode2()` after
// the name when !IsEntrusted()).
func TestInteractionMiniGameRoomBytesV95(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 95, 1)
	avatarBytes := singleEquipAvatar().Encode(l, ctx)(nil)

	got := test.Encode(t, ctx, miniGameRoomFixtureInput().Encode, nil)
	want := miniGameRoomWant(avatarBytes, true) // v95: per-avatar jobCode present

	if !bytes.Equal(got, want) {
		t.Fatalf("byte mismatch:\n got  %x\n want %x", got, want)
	}
}

// TestInteractionMiniGameRoomTournamentByte asserts the trailing round byte is
// written iff tournament is true (COmokDlg::OnEnterResult: `if (tournament)
// round = Decode1()`).
func TestInteractionMiniGameRoomTournamentByte(t *testing.T) {
	ctx := test.CreateContext("GMS", 83, 1)
	avatar := singleEquipAvatar()
	players := []MiniGameRoomPlayer{{Slot: 0, Avatar: avatar, Name: "Owner"}}

	noTourney := test.Encode(t, ctx, NewInteractionMiniGameRoom(5, interaction.OmokRoomType, 2, 0, players, "R", 0, false, 0).Encode, nil)
	tourney := test.Encode(t, ctx, NewInteractionMiniGameRoom(5, interaction.OmokRoomType, 2, 0, players, "R", 0, true, 7).Encode, nil)

	if len(tourney) != len(noTourney)+1 {
		t.Fatalf("tournament round byte: tourney len %d, non-tourney len %d (want +1)", len(tourney), len(noTourney))
	}
	if tourney[len(tourney)-1] != 7 {
		t.Fatalf("round byte: got %d, want 7", tourney[len(tourney)-1])
	}
	if tourney[len(tourney)-2] != 1 {
		t.Fatalf("tournament bool byte: got %d, want 1", tourney[len(tourney)-2])
	}
}

func TestInteractionMiniGameRoomRoundTrip(t *testing.T) {
	avatar := singleEquipAvatar()
	players := []MiniGameRoomPlayer{
		{Slot: 0, Avatar: avatar, Name: "Owner", Record: interaction.GameRecord{Unknown: 1, Wins: 10, Ties: 2, Losses: 3, Points: 500}},
		{Slot: 1, Avatar: avatar, Name: "Guest", Record: interaction.GameRecord{Unknown: 1, Wins: 4, Ties: 1, Losses: 6, Points: 250}},
	}
	input := NewInteractionMiniGameRoom(5, interaction.MatchCardRoomType, 2, 1, players, "Cards", 2, false, 0)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, (&InteractionMiniGameRoom{}).Decode, nil)
		})
	}
}
