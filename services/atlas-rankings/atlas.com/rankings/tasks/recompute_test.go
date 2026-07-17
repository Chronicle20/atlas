package tasks

import (
	"context"
	"errors"
	"testing"
	"time"

	"atlas-rankings/ranking"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type fakeProcessor struct {
	due          bool
	dueErr       error
	recomputeErr error
	recomputed   *int
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
