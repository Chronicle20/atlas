package ingest

import (
	"atlas-data/ingestrun"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"

	env "github.com/Chronicle20/atlas/libs/atlas-env"
	redis "github.com/Chronicle20/atlas/libs/atlas-redis"
)

func testSink(t *testing.T, runId string) (*redisSink, *miniredis.Miniredis, string) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	suffix := ingestrun.KeySuffix("shared", "GMS", 83, 1)
	l := logrus.New()
	l.SetLevel(logrus.PanicLevel)
	return newRedisSink(l, ingestrun.NewRunRegistry(rdb), suffix, runId), mr, suffix
}

func readRecord(t *testing.T, s *redisSink) ingestrun.Record {
	t.Helper()
	rec, err := s.reg.Get(context.Background(), env.Self(), s.key)
	if err != nil {
		t.Fatalf("read record: %v", err)
	}
	return rec
}

func seedFor(runId string, at time.Time) ingestrun.Record {
	return ingestrun.NewRecord(runId, "", "shared", "GMS", "83.1", "", at, []string{"STRING", "MAP"})
}

// The REST pod normally wrote the record already; Init must adopt it —
// preserving runId, jobName and startedAt — and only seed the roster.
func TestSinkInitAdoptsExistingRecord(t *testing.T) {
	s, _, _ := testSink(t, "run-1")
	ctx := context.Background()
	created := time.Date(2026, 8, 8, 10, 0, 0, 0, time.UTC)

	existing := ingestrun.NewRecord("run-1", "job-1", "shared", "GMS", "83.1", "", created, nil)
	if err := s.reg.PutWithTTL(ctx, env.Self(), s.key, existing, ingestrun.RecordTTL); err != nil {
		t.Fatal(err)
	}

	s.Init(ctx, seedFor("run-1", time.Now().UTC()), []string{"STRING", "MAP"}, time.Now().UTC())

	got := readRecord(t, s)
	if !got.StartedAt.Equal(created) {
		t.Fatalf("startedAt = %v, want the REST pod's %v", got.StartedAt, created)
	}
	if got.JobName != "job-1" || got.RunId != "run-1" {
		t.Fatalf("identity not preserved: %+v", got)
	}
	if len(got.Workers) != 2 {
		t.Fatalf("roster size = %d, want 2", len(got.Workers))
	}
	if got.Phase != ingestrun.PhaseRunning {
		t.Fatalf("phase = %s, want running", got.Phase)
	}
}

// Redis may have lost the record between Job creation and pod start.
func TestSinkInitSeedsWhenAbsent(t *testing.T) {
	s, _, _ := testSink(t, "run-1")
	ctx := context.Background()
	now := time.Date(2026, 8, 8, 11, 0, 0, 0, time.UTC)

	s.Init(ctx, seedFor("run-1", now), []string{"STRING", "MAP"}, now)

	got := readRecord(t, s)
	if got.RunId != "run-1" || got.Phase != ingestrun.PhaseRunning || len(got.Workers) != 2 {
		t.Fatalf("seeded record wrong: %+v", got)
	}
}

func TestSinkWorkerTransitions(t *testing.T) {
	s, _, _ := testSink(t, "run-1")
	ctx := context.Background()
	now := time.Now().UTC()
	s.Init(ctx, seedFor("run-1", now), []string{"STRING", "MAP"}, now)

	s.WorkerStarted(ctx, "STRING")
	if got := readRecord(t, s); got.Workers[0].State != ingestrun.WorkerRunning {
		t.Fatalf("STRING = %s, want running", got.Workers[0].State)
	}

	s.WorkerFinished(ctx, "STRING", nil, false)
	if got := readRecord(t, s); got.Workers[0].State != ingestrun.WorkerSucceeded {
		t.Fatalf("STRING = %s, want succeeded", got.Workers[0].State)
	}

	s.WorkerFinished(ctx, "MAP", nil, true)
	if got := readRecord(t, s); got.Workers[1].State != ingestrun.WorkerSkipped {
		t.Fatalf("MAP = %s, want skipped", got.Workers[1].State)
	}

	s.Finish(ctx, nil, time.Now().UTC())
	got := readRecord(t, s)
	if got.Phase != ingestrun.PhaseSucceeded {
		t.Fatalf("phase = %s, want succeeded", got.Phase)
	}
	if got.CompleteCount() != 2 {
		t.Fatalf("CompleteCount = %d, want 2 (skipped counts)", got.CompleteCount())
	}
}

func TestSinkFinishRecordsFailure(t *testing.T) {
	s, _, _ := testSink(t, "run-1")
	ctx := context.Background()
	now := time.Now().UTC()
	s.Init(ctx, seedFor("run-1", now), []string{"STRING", "MAP"}, now)
	s.WorkerFinished(ctx, "STRING", errors.New("boom"), false)

	s.Finish(ctx, errors.New("STRING open String.wz: boom"), time.Now().UTC())

	got := readRecord(t, s)
	if got.Phase != ingestrun.PhaseFailed {
		t.Fatalf("phase = %s, want failed", got.Phase)
	}
	if got.Reason == "" {
		t.Fatal("failed run has no reason")
	}
	if got.Workers[0].State != ingestrun.WorkerFailed || got.Workers[0].Error != "boom" {
		t.Fatalf("worker failure not recorded: %+v", got.Workers[0])
	}
}

// When a worker fails the errgroup cancels the context; the terminal write
// must still land, or the run is stuck at `running` forever.
func TestSinkFinishSurvivesCancelledContext(t *testing.T) {
	s, _, _ := testSink(t, "run-1")
	now := time.Now().UTC()
	s.Init(context.Background(), seedFor("run-1", now), []string{"STRING"}, now)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s.Finish(ctx, errors.New("boom"), time.Now().UTC())

	if got := readRecord(t, s); got.Phase != ingestrun.PhaseFailed {
		t.Fatalf("phase = %s, want failed (write must not depend on a live ctx)", got.Phase)
	}
}

// An operator re-triggering an ingest while this pod is alive replaces the
// record. A stale pod's writes must be dropped, not overwrite the new run.
func TestSinkDropsWritesForASupersededRun(t *testing.T) {
	s, _, _ := testSink(t, "run-OLD")
	ctx := context.Background()
	now := time.Now().UTC()

	fresh := ingestrun.NewRecord("run-NEW", "job-2", "shared", "GMS", "83.1", "", now, []string{"STRING"})
	if err := s.reg.PutWithTTL(ctx, env.Self(), s.key, fresh, ingestrun.RecordTTL); err != nil {
		t.Fatal(err)
	}

	s.WorkerFinished(ctx, "STRING", nil, false)
	s.Finish(ctx, nil, time.Now().UTC())

	got := readRecord(t, s)
	if got.RunId != "run-NEW" {
		t.Fatalf("runId = %s, want run-NEW", got.RunId)
	}
	if got.Phase != ingestrun.PhaseRunning {
		t.Fatalf("phase = %s: the stale pod stamped a terminal phase on the new run", got.Phase)
	}
	if got.Workers[0].State != ingestrun.WorkerPending {
		t.Fatalf("STRING = %s: the stale pod wrote into the new run", got.Workers[0].State)
	}
}

// A pod scheduled so late that Redis already holds a newer (possibly already
// terminal) run's record must not have its own Init revert that record back
// to running — Init is a mutation like any other and must honor the same
// superseded-pod guard as WorkerStarted/WorkerFinished/Finish (design §3.1).
func TestSinkInitDropsWritesForASupersededRun(t *testing.T) {
	s, _, _ := testSink(t, "run-OLD")
	ctx := context.Background()
	now := time.Now().UTC()

	fresh := ingestrun.NewRecord("run-NEW", "job-2", "shared", "GMS", "83.1", "", now, []string{"STRING"})
	fresh = fresh.WithPhase(ingestrun.PhaseSucceeded, now, "")
	if err := s.reg.PutWithTTL(ctx, env.Self(), s.key, fresh, ingestrun.RecordTTL); err != nil {
		t.Fatal(err)
	}

	s.Init(ctx, seedFor("run-OLD", now), []string{"STRING", "MAP"}, now)

	got := readRecord(t, s)
	if got.RunId != "run-NEW" {
		t.Fatalf("runId = %s, want run-NEW", got.RunId)
	}
	if got.Phase != ingestrun.PhaseSucceeded {
		t.Fatalf("phase = %s: the stale pod's Init reverted a completed run to running", got.Phase)
	}
	if len(got.Workers) != 1 || got.Workers[0].Name != "STRING" {
		t.Fatalf("roster mutated by a stale pod's Init: %+v", got.Workers)
	}
}

// FR-2.5: a Redis outage is warn-logged telemetry, never an ingest failure.
func TestSinkSurvivesRedisOutage(t *testing.T) {
	s, mr, _ := testSink(t, "run-1")
	ctx := context.Background()
	now := time.Now().UTC()
	s.Init(ctx, seedFor("run-1", now), []string{"STRING"}, now)
	mr.Close()

	// None of these return an error, and none may panic.
	s.WorkerStarted(ctx, "STRING")
	s.WorkerFinished(ctx, "STRING", nil, false)
	s.Finish(ctx, nil, time.Now().UTC())
}

// The TTL must be refreshed on every write — a record that lost its expiry
// would live forever (the reason UpdateWithTTL exists).
func TestSinkWritesKeepTheTTL(t *testing.T) {
	s, mr, suffix := testSink(t, "run-1")
	ctx := context.Background()
	now := time.Now().UTC()
	s.Init(ctx, seedFor("run-1", now), []string{"STRING"}, now)
	s.WorkerStarted(ctx, "STRING")

	key := redis.KeyPrefix() + ":" + ingestrun.Namespace + ":" + suffix + ingestrun.RunKeySuffix
	if ttl := mr.TTL(key); ttl <= 0 {
		t.Fatalf("TTL = %v, want > 0", ttl)
	}
}
