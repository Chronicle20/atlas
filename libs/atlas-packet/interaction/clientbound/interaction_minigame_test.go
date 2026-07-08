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
