package ingestrun

import (
	"testing"
	"time"
)

func names() []string { return []string{"STRING", "MAP", "ITEM"} }

func TestNewRecordSeedsRosterPending(t *testing.T) {
	start := time.Date(2026, 8, 8, 10, 0, 0, 0, time.UTC)
	rec := NewRecord("run-1", "job-1", "tenants/t1", "GMS", "83.1", "t1", start, names())

	if rec.Phase != PhaseRunning {
		t.Fatalf("phase = %s, want %s", rec.Phase, PhaseRunning)
	}
	if len(rec.Workers) != 3 {
		t.Fatalf("roster size = %d, want 3", len(rec.Workers))
	}
	for _, w := range rec.Workers {
		if w.State != WorkerPending {
			t.Fatalf("worker %s state = %s, want %s", w.Name, w.State, WorkerPending)
		}
	}
	if rec.CompleteCount() != 0 {
		t.Fatalf("CompleteCount = %d, want 0", rec.CompleteCount())
	}
}

func TestKeySuffix(t *testing.T) {
	if got := KeySuffix("tenants/t1", "GMS", 83, 1); got != "tenants/t1:GMS:83.1" {
		t.Fatalf("got %q", got)
	}
	if got := KeySuffix("shared", "JMS", 185, 1); got != "shared:JMS:185.1" {
		t.Fatalf("got %q", got)
	}
}

func TestWorkerTransitions(t *testing.T) {
	start := time.Date(2026, 8, 8, 10, 0, 0, 0, time.UTC)
	rec := NewRecord("run-1", "job-1", "shared", "GMS", "83.1", "", start, names())

	rec = rec.WithWorkerRunning("STRING", start.Add(time.Second))
	if rec.Workers[0].State != WorkerRunning || rec.Workers[0].StartedAt == nil {
		t.Fatalf("STRING not running with a startedAt: %+v", rec.Workers[0])
	}

	rec = rec.WithWorkerTerminal("STRING", WorkerSucceeded, start.Add(2*time.Second), "")
	if rec.Workers[0].State != WorkerSucceeded || rec.Workers[0].FinishedAt == nil {
		t.Fatalf("STRING not succeeded with a finishedAt: %+v", rec.Workers[0])
	}
	if rec.Workers[0].Error != "" {
		t.Fatalf("succeeded worker carries error %q", rec.Workers[0].Error)
	}

	rec = rec.WithWorkerTerminal("MAP", WorkerFailed, start.Add(3*time.Second), "boom")
	if rec.Workers[1].State != WorkerFailed || rec.Workers[1].Error != "boom" {
		t.Fatalf("MAP not failed with error: %+v", rec.Workers[1])
	}
}

// A worker whose category is genuinely absent from a monolithic Data.wz counts
// as complete and must not stop the run reaching `succeeded` (PRD FR-1.5).
func TestSkippedCountsCompleteAndDoesNotBlockSucceeded(t *testing.T) {
	start := time.Date(2026, 8, 8, 10, 0, 0, 0, time.UTC)
	rec := NewRecord("run-1", "job-1", "shared", "GMS", "83.1", "", start, names())
	rec = rec.WithWorkerTerminal("STRING", WorkerSucceeded, start, "")
	rec = rec.WithWorkerTerminal("MAP", WorkerSkipped, start, "")
	rec = rec.WithWorkerTerminal("ITEM", WorkerSucceeded, start, "")

	if rec.CompleteCount() != 3 {
		t.Fatalf("CompleteCount = %d, want 3 (skipped counts as complete)", rec.CompleteCount())
	}
	rec = rec.WithPhase(PhaseSucceeded, start.Add(time.Minute), "")
	if rec.Phase != PhaseSucceeded {
		t.Fatalf("phase = %s, want %s", rec.Phase, PhaseSucceeded)
	}
	if rec.FinishedAt == nil {
		t.Fatal("terminal phase left FinishedAt nil")
	}
}

func TestWithPhaseNonTerminalLeavesFinishedAtNil(t *testing.T) {
	start := time.Date(2026, 8, 8, 10, 0, 0, 0, time.UTC)
	rec := NewRecord("run-1", "job-1", "shared", "GMS", "83.1", "", start, names())
	rec = rec.WithPhase(PhaseRunning, start.Add(time.Minute), "")
	if rec.FinishedAt != nil {
		t.Fatal("running phase set FinishedAt")
	}
	if !rec.StartedAt.Equal(start) {
		t.Fatal("WithPhase overwrote StartedAt")
	}
}

// The mutators run inside an optimistic-lock closure that may execute many
// times against the same input. They must not mutate the caller's slice.
func TestTransitionsDoNotMutateReceiver(t *testing.T) {
	start := time.Date(2026, 8, 8, 10, 0, 0, 0, time.UTC)
	orig := NewRecord("run-1", "job-1", "shared", "GMS", "83.1", "", start, names())

	_ = orig.WithWorkerRunning("STRING", start)
	_ = orig.WithWorkerTerminal("MAP", WorkerFailed, start, "boom")
	_ = orig.WithPhase(PhaseFailed, start, "boom")

	for _, w := range orig.Workers {
		if w.State != WorkerPending {
			t.Fatalf("receiver mutated: %s is %s", w.Name, w.State)
		}
	}
	if orig.Phase != PhaseRunning || orig.FinishedAt != nil {
		t.Fatalf("receiver phase mutated: %+v", orig.Phase)
	}
}

// A record written by an older REST pod may not know about a worker the ingest
// pod is running. Transitions must record it rather than drop it.
func TestUnknownWorkerIsAppended(t *testing.T) {
	start := time.Date(2026, 8, 8, 10, 0, 0, 0, time.UTC)
	rec := NewRecord("run-1", "job-1", "shared", "GMS", "83.1", "", start, names())
	rec = rec.WithWorkerRunning("BRAND_NEW", start)
	if len(rec.Workers) != 4 || rec.Workers[3].Name != "BRAND_NEW" {
		t.Fatalf("unknown worker not appended: %+v", rec.Workers)
	}
}

func TestWithRosterAddsOnlyMissingNames(t *testing.T) {
	start := time.Date(2026, 8, 8, 10, 0, 0, 0, time.UTC)
	rec := NewRecord("run-1", "job-1", "shared", "GMS", "83.1", "", start, names())
	rec = rec.WithWorkerTerminal("STRING", WorkerSucceeded, start, "")

	rec = rec.WithRoster([]string{"STRING", "MAP", "ITEM", "COMMODITY"})
	if len(rec.Workers) != 4 {
		t.Fatalf("roster size = %d, want 4", len(rec.Workers))
	}
	if rec.Workers[0].State != WorkerSucceeded {
		t.Fatal("WithRoster reset an already-terminal worker")
	}
	if rec.Workers[3].Name != "COMMODITY" || rec.Workers[3].State != WorkerPending {
		t.Fatalf("new roster entry wrong: %+v", rec.Workers[3])
	}
}

func TestIsTerminal(t *testing.T) {
	cases := map[Phase]bool{
		PhaseNone: false, PhaseRunning: false, PhaseUnknown: false,
		PhaseSucceeded: true, PhaseFailed: true, PhaseStuck: true,
	}
	for p, want := range cases {
		if got := (Record{Phase: p}).IsTerminal(); got != want {
			t.Fatalf("IsTerminal(%s) = %v, want %v", p, got, want)
		}
	}
}
