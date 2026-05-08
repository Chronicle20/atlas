package extraction

import (
	"atlas-wz-extractor/extraction/job"
	"atlas-wz-extractor/extraction/lock"
	mext "atlas-wz-extractor/kafka/message/extraction"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
)

type fakeProcessor struct {
	calls   int
	failOn  string
	failErr error
}

func (f *fakeProcessor) ExtractUnit(l logrus.FieldLogger, ctx context.Context, wzFile string, xmlOnly, imagesOnly bool) error {
	f.calls++
	if wzFile == f.failOn {
		return f.failErr
	}
	return nil
}

func newRedis(t *testing.T) *goredis.Client {
	t.Helper()
	mr := miniredis.RunT(t)
	return goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
}

func TestHandler_HappyPath_FinalizesJob(t *testing.T) {
	ctx := context.Background()
	c := newRedis(t)
	store := job.NewStore(c)
	tl := lock.NewTenantLock(c, time.Minute)

	// seed: one-unit job, lock held by that job
	now := time.Now().UTC().Truncate(time.Second)
	j := job.NewJobBuilder().SetId("J").SetTenantId("T").SetRegion("GMS").
		SetMajorVersion(83).SetMinorVersion(1).
		SetStatus(job.JobRunning).SetUnitsTotal(1).SetCreatedAt(now).SetUpdatedAt(now).Build()
	if err := store.Create(ctx, j, []job.Unit{job.NewUnitBuilder().SetWzFile("Map.wz").SetStatus(job.UnitPending).Build()}, 3600); err != nil {
		t.Fatal(err)
	}
	lockKey := job.LockKey("T", "GMS", 83, 1)
	if _, err := tl.Acquire(ctx, lockKey, "J"); err != nil {
		t.Fatal(err)
	}

	fp := &fakeProcessor{}
	l, _ := test.NewNullLogger()
	h := handleStartExtractionUnit(fp, store, tl)

	h(l, ctx, command[startExtractionUnitBody]{
		Type: mext.CommandStartExtractionUnit,
		Body: startExtractionUnitBody{JobId: "J", WzFile: "Map.wz"},
	})

	gotJob, units, err := store.Get(ctx, "J")
	if err != nil {
		t.Fatal(err)
	}
	if gotJob.Status() != job.JobCompleted {
		t.Fatalf("status: %s", gotJob.Status())
	}
	if len(units) != 1 || units[0].Status() != job.UnitSucceeded {
		t.Fatalf("unit status: %+v", units)
	}
	// lock was released
	heldBy := c.Get(ctx, lockKey).Val()
	if heldBy != "" {
		t.Fatalf("lock should be released; still held by %q", heldBy)
	}
}

func TestHandler_RedeliverySkipsWork(t *testing.T) {
	ctx := context.Background()
	c := newRedis(t)
	store := job.NewStore(c)
	tl := lock.NewTenantLock(c, time.Minute)

	now := time.Now().UTC().Truncate(time.Second)
	j := job.NewJobBuilder().SetId("J2").SetTenantId("T").SetRegion("GMS").
		SetMajorVersion(83).SetMinorVersion(1).
		SetStatus(job.JobRunning).SetUnitsTotal(1).SetCreatedAt(now).SetUpdatedAt(now).Build()
	_ = store.Create(ctx, j, []job.Unit{job.NewUnitBuilder().SetWzFile("Map.wz").SetStatus(job.UnitSucceeded).SetCompletedAt(now).Build()}, 3600)
	_, _ = store.MarkJobTerminal(ctx, "J2", job.JobCompleted)

	fp := &fakeProcessor{}
	l, _ := test.NewNullLogger()
	h := handleStartExtractionUnit(fp, store, tl)

	h(l, ctx, command[startExtractionUnitBody]{
		Type: mext.CommandStartExtractionUnit,
		Body: startExtractionUnitBody{JobId: "J2", WzFile: "Map.wz"},
	})

	if fp.calls != 0 {
		t.Fatalf("expected ExtractUnit to be skipped on redelivery, got %d calls", fp.calls)
	}
}

func TestHandler_FailedUnit_MarksJobFailed(t *testing.T) {
	ctx := context.Background()
	c := newRedis(t)
	store := job.NewStore(c)
	tl := lock.NewTenantLock(c, time.Minute)

	now := time.Now().UTC().Truncate(time.Second)
	j := job.NewJobBuilder().SetId("J3").SetTenantId("T").SetRegion("GMS").
		SetMajorVersion(83).SetMinorVersion(1).
		SetStatus(job.JobRunning).SetUnitsTotal(1).SetCreatedAt(now).SetUpdatedAt(now).Build()
	_ = store.Create(ctx, j, []job.Unit{job.NewUnitBuilder().SetWzFile("Bad.wz").SetStatus(job.UnitPending).Build()}, 3600)
	lockKey := job.LockKey("T", "GMS", 83, 1)
	_, _ = tl.Acquire(ctx, lockKey, "J3")

	fp := &fakeProcessor{failOn: "Bad.wz", failErr: errors.New("boom")}
	l, _ := test.NewNullLogger()
	h := handleStartExtractionUnit(fp, store, tl)

	h(l, ctx, command[startExtractionUnitBody]{
		Type: mext.CommandStartExtractionUnit,
		Body: startExtractionUnitBody{JobId: "J3", WzFile: "Bad.wz"},
	})

	gotJob, _, _ := store.Get(ctx, "J3")
	if gotJob.Status() != job.JobFailed {
		t.Fatalf("expected JobFailed, got %s", gotJob.Status())
	}
}
