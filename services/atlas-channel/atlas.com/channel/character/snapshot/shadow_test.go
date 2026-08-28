package snapshot

import (
	"atlas-channel/character"
	"atlas-channel/character/buff"
	"atlas-channel/character/buff/stat"
	"atlas-channel/inventory"
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/sirupsen/logrus"

	charconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
)

func TestShadow_DisabledByDefault(t *testing.T) {
	tests := []struct {
		name string
		fn   func(t *testing.T)
	}{
		{"shadow is off by default: no extra core fetch on a full hit", func(t *testing.T) {
			resetRegistryForTest(t)
			resetShadowForTest(t)
			p, _ := newTestProcessor(t)
			counts := installFetchSeams(t, 7)
			if _, err := p.Get(7); err != nil {
				t.Fatalf("populate: %v", err)
			}
			if _, err := p.Get(7); err != nil { // full hit — the only sample point
				t.Fatalf("hit: %v", err)
			}
			waitForShadowDrain(t)
			if counts.core != 1 {
				t.Fatalf("shadow must be off by default: %+v", counts)
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, tt.fn)
	}
}

func TestShadow_SamplesAndCountsDivergence(t *testing.T) {
	tests := []struct {
		name string
		fn   func(t *testing.T)
	}{
		{"rate=1.0 shadow-fetches on a full hit and records exactly one core divergence", func(t *testing.T) {
			resetRegistryForTest(t)
			resetShadowForTest(t)
			t.Setenv("CHAR_SNAPSHOT_SHADOW_SAMPLE_RATE", "1.0")
			p, tm := newTestProcessor(t)
			counts := installFetchSeams(t, 7)
			if _, err := p.Get(7); err != nil {
				t.Fatalf("populate: %v", err)
			}
			divergenceBefore := testutil.ToFloat64(snapshotDivergenceTotal.WithLabelValues(tm.Id().String(), componentCore))

			// Make the REST projection diverge on level.
			var diverged atomic.Int32
			prevCore := coreFetchFn
			coreFetchFn = func(l logrus.FieldLogger, ctx context.Context, id uint32) (character.Model, error) {
				diverged.Add(1)
				return character.CloneModel(testCore(t, id)).SetLevel(99).MustBuild(), nil
			}
			t.Cleanup(func() { coreFetchFn = prevCore })

			if _, err := p.Get(7); err != nil {
				t.Fatalf("hit: %v", err)
			}
			waitForShadowDrain(t)
			if diverged.Load() == 0 {
				t.Fatalf("rate=1.0 must shadow-fetch on a full hit")
			}
			_ = counts
			divergenceAfter := testutil.ToFloat64(snapshotDivergenceTotal.WithLabelValues(tm.Id().String(), componentCore))
			if divergenceAfter-divergenceBefore != 1 {
				t.Fatalf("shadow divergence must record the core component once: before=%v after=%v", divergenceBefore, divergenceAfter)
			}
			// Divergence is observable via compareProjection directly:
			inv, _, _ := testInventory(t, 7)
			snapM := testCore(t, 7).SetInventory(inv).SetSkills(nil)
			restM := character.CloneModel(testCore(t, 7)).SetLevel(99).MustBuild().SetInventory(inv).SetSkills(nil)
			div := compareProjection(snapM, restM, nil, nil)
			if len(div) != 1 || div[0] != componentCore {
				t.Fatalf("level divergence must flag core: %v", div)
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, tt.fn)
	}
}

// TestShadow_BuffsDivergenceFires proves both call sites now thread real
// served buffs into maybeShadow (bug-shadow-buffs-dead-code part 1): before
// the fix, servedBuffs was always nil at both call sites, so
// compareProjection's buff branch could never execute and the buffs
// divergence metric could never fire.
func TestShadow_BuffsDivergenceFires(t *testing.T) {
	tests := []struct {
		name string
		fn   func(t *testing.T)
	}{
		{"rate=1.0 shadow-fetches on a full hit and records exactly one buffs divergence when the projectile-gate buff drops out server-side", func(t *testing.T) {
			resetRegistryForTest(t)
			resetShadowForTest(t)
			t.Setenv("CHAR_SNAPSHOT_SHADOW_SAMPLE_RATE", "1.0")
			p, tm := newTestProcessor(t)
			installFetchSeams(t, 7)

			// Populate core/inv/skills (Get() does not touch buffs; buffs
			// are seeded directly through the registry, the same seam
			// GetBuffs()'s own backfill uses).
			if _, err := p.Get(7); err != nil {
				t.Fatalf("populate: %v", err)
			}

			soulArrow := buff.NewBuff(3111004, 20, 60000,
				[]stat.Model{stat.NewStat(string(charconst.TemporaryStatTypeSoulArrow), 1)},
				time.Now(), time.Now().Add(time.Minute), false)
			r := GetRegistry()
			bv := r.View(tm, 7)
			if !r.BackfillBuffs(tm, 7, []buff.Model{soulArrow}, bv.BuffsGen) {
				t.Fatalf("seed backfill rejected")
			}

			divergenceBefore := testutil.ToFloat64(snapshotDivergenceTotal.WithLabelValues(tm.Id().String(), componentBuffs))

			// The REST projection no longer carries the gate buff.
			// installFetchSeams already registered a cleanup that restores
			// the pre-test seams, so reassigning here is safe.
			buffsFetchFn = func(l logrus.FieldLogger, ctx context.Context, id uint32) ([]buff.Model, error) {
				return []buff.Model{}, nil
			}

			if _, err := p.Get(7); err != nil { // full hit (fast path) — servedBuffs must now be non-nil
				t.Fatalf("hit: %v", err)
			}
			waitForShadowDrain(t)

			divergenceAfter := testutil.ToFloat64(snapshotDivergenceTotal.WithLabelValues(tm.Id().String(), componentBuffs))
			if divergenceAfter-divergenceBefore != 1 {
				t.Fatalf("shadow divergence must record the buffs component once: before=%v after=%v", divergenceBefore, divergenceAfter)
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, tt.fn)
	}
}

func TestCompareProjection_PositionToleranceBanded(t *testing.T) {
	inv, _, _ := testInventory(t, 7)
	a := testCore(t, 7).SetInventory(inv).SetSkills(nil)

	tests := []struct {
		name        string
		dx          int16
		wantDiverge []string
	}{
		{"within band: no divergence", 90, nil},
		{"beyond band: position divergence", 500, []string{componentPosition}},
		// dx == positionToleranceBand is within the band (shadow.go compares
		// with strict > / <), so it must not diverge.
		{"exact boundary: dx == positionToleranceBand must not diverge", positionToleranceBand, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b := character.CloneModel(testCore(t, 7)).SetX(testCore(t, 7).X() + tc.dx).MustBuild().SetInventory(inv).SetSkills(nil)
			div := compareProjection(a, b, nil, nil)
			if len(div) != len(tc.wantDiverge) {
				t.Fatalf("divergence mismatch: got %v want %v", div, tc.wantDiverge)
			}
			for i, want := range tc.wantDiverge {
				if div[i] != want {
					t.Fatalf("divergence mismatch: got %v want %v", div, tc.wantDiverge)
				}
			}
		})
	}
}

// waitForShadowDrain waits for in-flight shadow goroutines (bounded by the
// semaphore) to finish.
func waitForShadowDrain(t *testing.T) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if shadowInFlight.Load() == 0 {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("shadow goroutines did not drain")
}

// resetShadowForTest resets the once-computed shadow sample rate so each
// test's CHAR_SNAPSHOT_SHADOW_SAMPLE_RATE (or its absence) is re-read.
func resetShadowForTest(t *testing.T) {
	t.Helper()
	shadowRateOnce = sync.Once{}
	shadowRateVal = 0
}

var _ = inventory.Model{} // keep import if unused by edits above
