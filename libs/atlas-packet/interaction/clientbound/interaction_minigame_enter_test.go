package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/interaction"
	testlog "github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// Game ENTER (mode 4, game-dialog shape). IDA-derived read order:
// CMiniRoomBaseDlg::OnEnterBase (v95 0x638f80: slot, AvatarLook, name,
// jobCode Decode2) then COmokDlg::OnEnter (v95 0x6812e0) /
// CMemoryGameDlg::OnEnter (v95 0x628980) GW_MiniGameRecord::Decode
// (0x4f2ad0 = 20 bytes). v83: no jobCode (same enterHasJobCode gate as the
// room-enter blob); record read is sub_4E42FC in COmokDlg::OnEnter =
// sub_6E3BCC @ 0x6e3bcc.
func miniGameEnterWant(avatarBytes []byte, withJobCode bool) []byte {
	var want []byte
	want = append(want, 4)                 // mode (ENTER)
	want = append(want, 1)                 // slot 1 (the visitor)
	want = append(want, avatarBytes...)    // AvatarLook blob
	want = append(want, ascii("Guest")...) // name
	if withJobCode {
		want = append(want, le16(230)...) // jobCode (m_anJobCode[1])
	}
	want = append(want, le32(1)...)   // Unknown
	want = append(want, le32(4)...)   // Wins
	want = append(want, le32(1)...)   // Ties
	want = append(want, le32(6)...)   // Losses
	want = append(want, le32(250)...) // Points
	return want
}

func miniGameEnterFixtureInput() InteractionMiniGameEnter {
	return NewInteractionMiniGameEnter(4, MiniGameRoomPlayer{
		Slot:    1,
		Avatar:  singleEquipAvatar(),
		Name:    "Guest",
		JobCode: 230,
		Record:  interaction.GameRecord{Unknown: 1, Wins: 4, Ties: 1, Losses: 6, Points: 250},
	})
}

// TestInteractionMiniGameEnterBytes asserts the EXACT gms_v83 wire sequence
// (no per-avatar jobCode on v83).
func TestInteractionMiniGameEnterBytes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 83, 1)
	avatarBytes := singleEquipAvatar().Encode(l, ctx)(nil)

	got := test.Encode(t, ctx, miniGameEnterFixtureInput().Encode, nil)
	want := miniGameEnterWant(avatarBytes, false)

	if !bytes.Equal(got, want) {
		t.Fatalf("byte mismatch:\n got  %x\n want %x", got, want)
	}
}

// TestInteractionMiniGameEnterBytesV95 asserts the EXACT gms_v95 wire
// sequence: identical to v83 plus the trailing uint16 jobCode after the name
// (v95 OnEnterBase @0x638f80: `m_anJobCode[v4] = Decode2()`).
func TestInteractionMiniGameEnterBytesV95(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 95, 1)
	avatarBytes := singleEquipAvatar().Encode(l, ctx)(nil)

	got := test.Encode(t, ctx, miniGameEnterFixtureInput().Encode, nil)
	want := miniGameEnterWant(avatarBytes, true)

	if !bytes.Equal(got, want) {
		t.Fatalf("byte mismatch:\n got  %x\n want %x", got, want)
	}
}

func TestInteractionMiniGameEnterRoundTrip(t *testing.T) {
	input := miniGameEnterFixtureInput()
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, (&InteractionMiniGameEnter{}).Decode, nil)
		})
	}
}
