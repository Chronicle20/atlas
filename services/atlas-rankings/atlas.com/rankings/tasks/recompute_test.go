package tasks

import (
	"atlas-rankings/ranking"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type fakeProcessor struct {
	due            bool
	dueErr         error
	recomputeErr   error
	recomputed     *int
	afterRecompute func()
}

func (f fakeProcessor) ByCharacterIdProvider(uint32) model.Provider[ranking.Model] {
	return func() (ranking.Model, error) { return ranking.Model{}, nil }
}
func (f fakeProcessor) GetByCharacterId(uint32) (ranking.Model, error) { return ranking.Model{}, nil }
func (f fakeProcessor) ByCharacterIdsProvider([]uint32) model.Provider[[]ranking.Model] {
	return func() ([]ranking.Model, error) { return nil, nil }
}
func (f fakeProcessor) GetByCharacterIds([]uint32) ([]ranking.Model, error) { return nil, nil }
func (f fakeProcessor) IsDue(time.Duration, time.Time) (bool, error)        { return f.due, f.dueErr }
func (f fakeProcessor) Recompute(time.Time) error {
	if f.recomputeErr != nil {
		return f.recomputeErr
	}
	*f.recomputed++
	if f.afterRecompute != nil {
		f.afterRecompute()
	}
	return nil
}

func (f fakeProcessor) WithCharacterSupplier(ranking.CharacterSupplier) ranking.Processor {
	return f
}

func testTenants(t *testing.T, n int) []tenant.Model {
	t.Helper()
	ts := make([]tenant.Model, 0, n)
	for i := 0; i < n; i++ {
		tm, err := tenant.Register(uuid.New(), "GMS", 83, 1)
		if err != nil {
			t.Fatalf("tenant: %v", err)
		}
		ts = append(ts, tm)
	}
	return ts
}

func TestRunSkipsFailingTenantAndContinues(t *testing.T) {
	ts := testTenants(t, 3)
	countA, countC := 0, 0

	task := &RecomputeTask{
		l:        logrus.New(),
		ctx:      context.Background(),
		interval: time.Minute,
		tenants:  func() ([]tenant.Model, error) { return ts, nil },
		intervalFor: func(context.Context, uuid.UUID) time.Duration {
			return time.Hour
		},
		processorFor: func(ctx context.Context) ranking.Processor {
			tm := tenant.MustFromContext(ctx)
			switch tm.Id() {
			case ts[0].Id():
				return fakeProcessor{due: true, recomputed: &countA}
			case ts[1].Id():
				return fakeProcessor{due: true, recomputeErr: errors.New("boom"), recomputed: new(int)}
			default:
				return fakeProcessor{due: true, recomputed: &countC}
			}
		},
	}

	task.Run()

	if countA != 1 || countC != 1 {
		t.Fatalf("tenant B failure must not stop others: A=%d C=%d", countA, countC)
	}
}

func TestRunSkipsNotDueTenants(t *testing.T) {
	ts := testTenants(t, 1)
	count := 0
	task := &RecomputeTask{
		l:        logrus.New(),
		ctx:      context.Background(),
		interval: time.Minute,
		tenants:  func() ([]tenant.Model, error) { return ts, nil },
		intervalFor: func(context.Context, uuid.UUID) time.Duration {
			return time.Hour
		},
		processorFor: func(context.Context) ranking.Processor {
			return fakeProcessor{due: false, recomputed: &count}
		},
	}
	task.Run()
	if count != 0 {
		t.Fatalf("not-due tenant must not recompute, got %d", count)
	}
}

func TestRunSkipsAllTenantsWhenContextAlreadyCancelled(t *testing.T) {
	ts := testTenants(t, 2)
	count := 0

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	task := &RecomputeTask{
		l:        logrus.New(),
		ctx:      ctx,
		interval: time.Minute,
		tenants:  func() ([]tenant.Model, error) { return ts, nil },
		intervalFor: func(context.Context, uuid.UUID) time.Duration {
			return time.Hour
		},
		processorFor: func(context.Context) ranking.Processor {
			t.Fatal("must not construct a processor when context is already cancelled")
			return nil
		},
	}

	task.Run()

	if count != 0 {
		t.Fatalf("already-cancelled context must process zero tenants, got %d", count)
	}
}

func TestRunStopsTenantLoopOnMidTickCancellation(t *testing.T) {
	ts := testTenants(t, 4)
	countBefore, countAfter := 0, 0

	ctx, cancel := context.WithCancel(context.Background())

	task := &RecomputeTask{
		l:        logrus.New(),
		ctx:      ctx,
		interval: time.Minute,
		tenants:  func() ([]tenant.Model, error) { return ts, nil },
		intervalFor: func(context.Context, uuid.UUID) time.Duration {
			return time.Hour
		},
		processorFor: func(pctx context.Context) ranking.Processor {
			tm := tenant.MustFromContext(pctx)
			switch tm.Id() {
			case ts[0].Id():
				// After the first tenant recomputes, cancel the context to
				// simulate a lease loss landing mid-tick.
				return fakeProcessor{due: true, recomputed: &countBefore, afterRecompute: cancel}
			default:
				return fakeProcessor{due: true, recomputed: &countAfter}
			}
		},
	}

	task.Run()

	if countBefore != 1 {
		t.Fatalf("first tenant should have recomputed before cancellation, got %d", countBefore)
	}
	if countAfter != 0 {
		t.Fatalf("cancellation mid-tick must stop the remaining tenants, got %d processed", countAfter)
	}
}

func TestRunSkipsTenantWhenIsDueErrorsAndContinues(t *testing.T) {
	ts := testTenants(t, 2)
	countB := 0

	task := &RecomputeTask{
		l:        logrus.New(),
		ctx:      context.Background(),
		interval: time.Minute,
		tenants:  func() ([]tenant.Model, error) { return ts, nil },
		intervalFor: func(context.Context, uuid.UUID) time.Duration {
			return time.Hour
		},
		processorFor: func(ctx context.Context) ranking.Processor {
			tm := tenant.MustFromContext(ctx)
			switch tm.Id() {
			case ts[0].Id():
				return fakeProcessor{dueErr: errors.New("cadence lookup failed"), recomputed: new(int)}
			default:
				return fakeProcessor{due: true, recomputed: &countB}
			}
		},
	}

	task.Run()

	if countB != 1 {
		t.Fatalf("IsDue error on one tenant must not stop the others, got %d", countB)
	}
}

func TestRunToleratesTenantEnumerationFailure(t *testing.T) {
	task := &RecomputeTask{
		l:        logrus.New(),
		ctx:      context.Background(),
		interval: time.Minute,
		tenants:  func() ([]tenant.Model, error) { return nil, errors.New("tenants down") },
		intervalFor: func(context.Context, uuid.UUID) time.Duration {
			return time.Hour
		},
		processorFor: func(context.Context) ranking.Processor {
			t.Fatal("must not construct a processor when enumeration fails")
			return nil
		},
	}
	task.Run() // must not panic
	if task.SleepTime() != time.Minute {
		t.Fatalf("SleepTime = %v, want 1m", task.SleepTime())
	}
}
