package snapshot

import (
	"atlas-channel/character"
	"atlas-channel/inventory"
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/sirupsen/logrus"
)

func TestShadow_DisabledByDefault(t *testing.T) {
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
}

func TestShadow_SamplesAndCountsDivergence(t *testing.T) {
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
}

func TestCompareProjection_PositionToleranceBanded(t *testing.T) {
	inv, _, _ := testInventory(t, 7)
	a := testCore(t, 7).SetInventory(inv).SetSkills(nil)
	// Within band: no divergence.
	b := character.CloneModel(testCore(t, 7)).SetX(testCore(t, 7).X() + 90).MustBuild().SetInventory(inv).SetSkills(nil)
	if div := compareProjection(a, b, nil, nil); len(div) != 0 {
		t.Fatalf("within-band position must not diverge: %v", div)
	}
	// Beyond band: position divergence.
	c := character.CloneModel(testCore(t, 7)).SetX(testCore(t, 7).X() + 500).MustBuild().SetInventory(inv).SetSkills(nil)
	if div := compareProjection(a, c, nil, nil); len(div) != 1 || div[0] != componentPosition {
		t.Fatalf("out-of-band position must diverge: %v", div)
	}
	// Exact boundary: dx == positionToleranceBand is within the band
	// (shadow.go compares with strict > / <), so it must not diverge.
	d := character.CloneModel(testCore(t, 7)).SetX(testCore(t, 7).X() + positionToleranceBand).MustBuild().SetInventory(inv).SetSkills(nil)
	if div := compareProjection(a, d, nil, nil); len(div) != 0 {
		t.Fatalf("dx exactly at positionToleranceBand must not diverge: %v", div)
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
