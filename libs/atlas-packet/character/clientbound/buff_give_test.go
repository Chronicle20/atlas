package clientbound

import (
	"bytes"
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func TestBuffGiveEmptyRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			cts := model.NewCharacterTemporaryStat()
			input := NewBuffGive(*cts)
			output := BuffGive{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}

// TestBuffGiveDiseaseTrailer pins that a BuffGive carrying a mob-applied
// disease (here SLOW) ends with Cosmic's giveDebuff trailer
// (Short(900) + Byte(1)) instead of the buff trailer (Short(0) + Byte(0)).
// Without this branch the v83 client gets the raw stat but skips the
// debuff icon and the flag-gated effects (WEAKEN jump-block, etc.).
func TestBuffGiveDiseaseTrailer(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	tn, _ := tenant.Create([16]byte{}, "GMS", 83, 1)
	cts := model.NewCharacterTemporaryStat()
	cts.AddStat(nil)(tn)(string(character.TemporaryStatTypeSlow), 126, 80, 2, time.Now().Add(15*time.Second))

	got := NewBuffGive(*cts).Encode(nil, ctx)(nil)
	if len(got) < 3 {
		t.Fatalf("encoded payload too short: %d bytes", len(got))
	}
	// Last 3 bytes: Short(900) + Byte(1) → 84 03 01.
	wantTail := []byte{0x84, 0x03, 0x01}
	tail := got[len(got)-3:]
	if !bytes.Equal(tail, wantTail) {
		t.Errorf("disease trailer: got %x want %x (full payload tail: %x)", tail, wantTail, got[max(0, len(got)-8):])
	}
}

// TestBuffGiveBuffTrailer pins that a BuffGive with only player buffs
// keeps the legacy trailer (Short(0) + Byte(0)) — guards against the
// disease branch accidentally swallowing buffs.
func TestBuffGiveBuffTrailer(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	tn, _ := tenant.Create([16]byte{}, "GMS", 83, 1)
	cts := model.NewCharacterTemporaryStat()
	cts.AddStat(nil)(tn)(string(character.TemporaryStatTypeInvincible), 2301003, 30, 20, time.Now().Add(5*time.Minute))

	got := NewBuffGive(*cts).Encode(nil, ctx)(nil)
	if len(got) < 3 {
		t.Fatalf("encoded payload too short: %d bytes", len(got))
	}
	wantTail := []byte{0x00, 0x00, 0x00}
	tail := got[len(got)-3:]
	if !bytes.Equal(tail, wantTail) {
		t.Errorf("buff trailer: got %x want %x", tail, wantTail)
	}
}

func TestBuffGiveForeignEmptyRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			cts := model.NewCharacterTemporaryStat()
			input := NewBuffGiveForeign(12345, *cts)
			output := BuffGiveForeign{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.CharacterId() != 12345 {
				t.Errorf("characterId: got %v, want %v", output.CharacterId(), 12345)
			}
		})
	}
}
