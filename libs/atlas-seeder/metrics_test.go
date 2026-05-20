package seeder

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestObserveSeederRun_IncrementsCounter(t *testing.T) {
	ResetMetricsForTest()
	ObserveSeederRun("atlas-test", "drops", "success", 0.5)
	ObserveSeederRun("atlas-test", "drops", "success", 0.25)
	got := testutil.ToFloat64(seederRunsTotal.WithLabelValues("atlas-test", "drops", "success"))
	if got != 2 {
		t.Fatalf("counter = %v, want 2", got)
	}
	histCount := testutil.CollectAndCount(seederDurationSeconds)
	if histCount == 0 {
		t.Fatalf("histogram not registered")
	}
}
