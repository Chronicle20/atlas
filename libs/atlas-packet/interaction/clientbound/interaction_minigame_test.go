package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// gms_v83/v95 (COmokDlg::OnPacket 0x6e37eb / 0x688b70, CMemoryGameDlg::OnPacket
// 0x64db30 / 0x634020): modes verified byte-identical on both tenants — see
// docs/tasks/task-133-miniroom-minigames/ida-notes.md §G5.
// packet-audit:verify packet=interaction/clientbound/InteractionMiniGameReady version=gms_v83 ida=0x6e4608
// packet-audit:verify packet=interaction/clientbound/InteractionMiniGameReady version=gms_v95 ida=0x684930
func TestInteractionMiniGameReadyRoundTrip(t *testing.T) {
	input := NewInteractionMiniGameReady(58)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, (&InteractionMiniGameReady{}).Decode, nil)
		})
	}
}

// packet-audit:verify packet=interaction/clientbound/InteractionMiniGameUnready version=gms_v83 ida=0x6e466e
// packet-audit:verify packet=interaction/clientbound/InteractionMiniGameUnready version=gms_v95 ida=0x6849c0
func TestInteractionMiniGameUnreadyRoundTrip(t *testing.T) {
	input := NewInteractionMiniGameUnready(59)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, (&InteractionMiniGameUnready{}).Decode, nil)
		})
	}
}

// packet-audit:verify packet=interaction/clientbound/InteractionMiniGameRequestTie version=gms_v83 ida=0x6e37eb
// packet-audit:verify packet=interaction/clientbound/InteractionMiniGameRequestTie version=gms_v95 ida=0x688b70
func TestInteractionMiniGameRequestTieRoundTrip(t *testing.T) {
	input := NewInteractionMiniGameRequestTie(50)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, (&InteractionMiniGameRequestTie{}).Decode, nil)
		})
	}
}

// packet-audit:verify packet=interaction/clientbound/InteractionMiniGameAnswerTie version=gms_v83 ida=0x6e37eb
// packet-audit:verify packet=interaction/clientbound/InteractionMiniGameAnswerTie version=gms_v95 ida=0x688b70
func TestInteractionMiniGameAnswerTieRoundTrip(t *testing.T) {
	input := NewInteractionMiniGameAnswerTie(51)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, (&InteractionMiniGameAnswerTie{}).Decode, nil)
		})
	}
}

// TestInteractionMiniGameSkipRoundTrip covers both `who` values: COmokDlg::OnTimeOver
// (v83 0x6e472e) stores `who` as the slot whose turn it now is (the next mover), not
// the skipper — see ida-notes.md §G5 SKIP section.
// packet-audit:verify packet=interaction/clientbound/InteractionMiniGameSkip version=gms_v83 ida=0x6e472e
// packet-audit:verify packet=interaction/clientbound/InteractionMiniGameSkip version=gms_v95 ida=0x67fac0
func TestInteractionMiniGameSkipRoundTrip(t *testing.T) {
	for _, who := range []byte{0x01, 0x00} {
		input := NewInteractionMiniGameSkip(63, who)
		for _, v := range test.Variants {
			t.Run(v.Name, func(t *testing.T) {
				ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
				test.RoundTrip(t, ctx, input.Encode, (&InteractionMiniGameSkip{}).Decode, nil)
			})
		}
	}
}

// gms_v83/v95 (COmokDlg::OnUserStart 0x6e469c / 0x684a00): first-mover byte
// semantics per ida-notes.md §G1 (first mover = slot != startByte).
// packet-audit:verify packet=interaction/clientbound/InteractionMiniGameStartOmok version=gms_v83 ida=0x6e469c
// packet-audit:verify packet=interaction/clientbound/InteractionMiniGameStartOmok version=gms_v95 ida=0x684a00
func TestInteractionMiniGameStartOmokRoundTrip(t *testing.T) {
	input := NewInteractionMiniGameStartOmok(61, 1)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, (&InteractionMiniGameStartOmok{}).Decode, nil)
		})
	}
}

// gms_v83 only (CMemoryGameDlg::OnUserStart 0x64e632) — ida-notes.md §G1/§G5
// record no v95 address for this handler; do not fabricate one.
// packet-audit:verify packet=interaction/clientbound/InteractionMiniGameStartMatchCards version=gms_v83 ida=0x64e632
func TestInteractionMiniGameStartMatchCardsRoundTrip(t *testing.T) {
	input := NewInteractionMiniGameStartMatchCards(61, 1, []uint32{0, 0, 1, 1, 2, 2, 3, 3, 4, 4, 5, 5})
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, (&InteractionMiniGameStartMatchCards{}).Decode, nil)
		})
	}
}

// gms_v83/v95 (COmokDlg::OnPutStoneChecker 0x6e3f5b / 0x6866a0).
// packet-audit:verify packet=interaction/clientbound/InteractionMiniGameMoveStone version=gms_v83 ida=0x6e3f5b
// packet-audit:verify packet=interaction/clientbound/InteractionMiniGameMoveStone version=gms_v95 ida=0x6866a0
func TestInteractionMiniGameMoveStoneRoundTrip(t *testing.T) {
	input := NewInteractionMiniGameMoveStone(64, 7, 8, 1)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, (&InteractionMiniGameMoveStone{}).Decode, nil)
		})
	}
}

// gms_v83/v95 (CMemoryGameDlg::OnTurnUpCard 0x64e1c1 / 0x62f060), turn byte == 1
// (first flip; forwarded to the opponent only — ida-notes.md §G5 SELECT_CARD).
// packet-audit:verify packet=interaction/clientbound/InteractionMiniGameCardSelectFirst version=gms_v83 ida=0x64e1c1
// packet-audit:verify packet=interaction/clientbound/InteractionMiniGameCardSelectFirst version=gms_v95 ida=0x62f060
func TestInteractionMiniGameCardSelectFirstRoundTrip(t *testing.T) {
	input := NewInteractionMiniGameCardSelectFirst(68, 3)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, (&InteractionMiniGameCardSelectFirst{}).Decode, nil)
		})
	}
}

// gms_v83/v95 (CMemoryGameDlg::OnTurnUpCard 0x64e1c1 / 0x62f060), turn byte == 0
// (second flip; forwarded to both players — ida-notes.md §G5 SELECT_CARD).
// packet-audit:verify packet=interaction/clientbound/InteractionMiniGameCardSelectSecond version=gms_v83 ida=0x64e1c1
// packet-audit:verify packet=interaction/clientbound/InteractionMiniGameCardSelectSecond version=gms_v95 ida=0x62f060
func TestInteractionMiniGameCardSelectSecondRoundTrip(t *testing.T) {
	input := NewInteractionMiniGameCardSelectSecond(68, 9, 3, 2)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, (&InteractionMiniGameCardSelectSecond{}).Decode, nil)
		})
	}
}
