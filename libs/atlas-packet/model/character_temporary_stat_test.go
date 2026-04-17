package model

import (
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func TestCTSForeignEmptyRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewCharacterTemporaryStat()
			output := NewCharacterTemporaryStat()
			pt.RoundTrip(t, ctx, input.EncodeForeign, output.DecodeForeign, nil)
			if len(output.stats) != 0 {
				t.Errorf("expected 0 decoded stats, got %d", len(output.stats))
			}
		})
	}
}

func TestCTSForeignSingleStatRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			tn, _ := tenant.Create([16]byte{}, v.Region, v.MajorVersion, v.MinorVersion)
			input := NewCharacterTemporaryStat()
			input.AddStat(nil)(tn)(string(character.TemporaryStatTypeSpeed), 2001002, 20, 10, time.Now().Add(time.Minute))
			output := NewCharacterTemporaryStat()
			pt.RoundTrip(t, ctx, input.EncodeForeign, output.DecodeForeign, nil)
			if len(output.stats) != 1 {
				t.Errorf("expected 1 decoded stat, got %d", len(output.stats))
			}
			if sv, ok := output.stats[character.TemporaryStatTypeSpeed]; ok {
				if sv.Value() != 20 {
					t.Errorf("speed value: got %d, want 20", sv.Value())
				}
			} else {
				t.Error("expected Speed stat to be present")
			}
		})
	}
}

func TestCTSForeignMultiStatRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			tn, _ := tenant.Create([16]byte{}, v.Region, v.MajorVersion, v.MinorVersion)
			input := NewCharacterTemporaryStat()
			// Byte writer
			input.AddStat(nil)(tn)(string(character.TemporaryStatTypeSpeed), 2001002, 20, 10, time.Now().Add(time.Minute))
			// Int writer
			input.AddStat(nil)(tn)(string(character.TemporaryStatTypeStun), 0, 1, 5, time.Now().Add(time.Minute))
			output := NewCharacterTemporaryStat()
			pt.RoundTrip(t, ctx, input.EncodeForeign, output.DecodeForeign, nil)
			if len(output.stats) != 2 {
				t.Errorf("expected 2 decoded stats, got %d", len(output.stats))
			}
		})
	}
}
