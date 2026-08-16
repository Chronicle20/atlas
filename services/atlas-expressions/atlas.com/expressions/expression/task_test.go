package expression

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

// envMarkerKey is a test-local context key -- deliberately not
// libs/atlas-env, since expression sits outside env-domain-guard's
// permitted import list (main.go, kafka/, rest/, socket/) and must not
// import atlas-env even from a test file.
type envMarkerKey string

func setupTaskTest(t *testing.T) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitRegistry(client)
}

func TestNewRevertTask(t *testing.T) {
	logger, _ := test.NewNullLogger()
	interval := 100 * time.Millisecond

	task := NewRevertTask(logger, interval, func(ctx context.Context) context.Context { return ctx })

	assert.NotNil(t, task)
}

func TestRevertTask_SleepTime(t *testing.T) {
	logger, _ := test.NewNullLogger()
	interval := 250 * time.Millisecond

	task := NewRevertTask(logger, interval, func(ctx context.Context) context.Context { return ctx })

	assert.Equal(t, interval, task.SleepTime())
}

func TestRevertTask_SleepTime_DifferentIntervals(t *testing.T) {
	logger, _ := test.NewNullLogger()

	testCases := []time.Duration{
		50 * time.Millisecond,
		100 * time.Millisecond,
		500 * time.Millisecond,
		1 * time.Second,
		5 * time.Second,
	}

	for _, interval := range testCases {
		task := NewRevertTask(logger, interval, func(ctx context.Context) context.Context { return ctx })
		assert.Equal(t, interval, task.SleepTime(), "SleepTime should return %v", interval)
	}
}

func TestRevertTask_Run_NoExpiredExpressions(t *testing.T) {
	setupTaskTest(t)

	logger, _ := test.NewNullLogger()
	task := NewRevertTask(logger, 100*time.Millisecond, func(ctx context.Context) context.Context { return ctx })

	ten := setupTestTenant(t)
	ctx := testCtx(ten)

	now := time.Now()
	GetRegistry().SetNowFunc(func() time.Time { return now })

	f := field.NewBuilder(0, 1, 100000000).Build()
	GetRegistry().add(ctx, 1000, f, 5)

	// Run should not panic and expression should still exist
	task.Run()

	_, found := GetRegistry().get(ctx, 1000)
	assert.True(t, found)
}

func TestRevertTask_Run_WithExpiredExpressions(t *testing.T) {
	setupTaskTest(t)

	logger, _ := test.NewNullLogger()
	task := NewRevertTask(logger, 100*time.Millisecond, func(ctx context.Context) context.Context { return ctx })

	ten := setupTestTenant(t)
	ctx := testCtx(ten)

	now := time.Now()
	GetRegistry().SetNowFunc(func() time.Time { return now })

	f := field.NewBuilder(0, 1, 100000000).Build()
	GetRegistry().add(ctx, 1000, f, 5)

	// Advance clock past TTL
	GetRegistry().SetNowFunc(func() time.Time { return now.Add(6 * time.Second) })

	task.Run()

	// Verify expression was removed
	_, found := GetRegistry().get(ctx, 1000)
	assert.False(t, found)
}

func TestRevertTask_Run_MixedExpiredAndNonExpired(t *testing.T) {
	setupTaskTest(t)

	logger, _ := test.NewNullLogger()
	task := NewRevertTask(logger, 100*time.Millisecond, func(ctx context.Context) context.Context { return ctx })

	ten := setupTestTenant(t)
	ctx := testCtx(ten)

	now := time.Now()
	GetRegistry().SetNowFunc(func() time.Time { return now })

	f := field.NewBuilder(0, 1, 100000000).Build()
	GetRegistry().add(ctx, 1000, f, 5)

	// Add second expression 3 seconds later
	GetRegistry().SetNowFunc(func() time.Time { return now.Add(3 * time.Second) })
	GetRegistry().add(ctx, 2000, f, 10)

	// Advance to 6 seconds - first expired, second not
	GetRegistry().SetNowFunc(func() time.Time { return now.Add(6 * time.Second) })

	task.Run()

	// Verify expired was removed
	_, found1 := GetRegistry().get(ctx, 1000)
	assert.False(t, found1)

	// Verify non-expired still exists
	_, found2 := GetRegistry().get(ctx, 2000)
	assert.True(t, found2)
}

// TestProcessExpiredAppliesEnvContextToRevert pins the review fix: this pod's
// own environment identity must be threaded onto each expired expression's
// per-tenant context before the revert emit. The existing Run-level tests
// above pass an identity envContext and would still pass if this were
// dropped -- decide() would then fail open per FR-1.8 and every live
// deployment, not just this pod's, would revert the expression.
func TestProcessExpiredAppliesEnvContextToRevert(t *testing.T) {
	l, _ := test.NewNullLogger()
	ten := setupTestTenant(t)
	f := field.NewBuilder(0, 1, 100000000).Build()
	m := NewModelBuilder(ten).SetCharacterId(1000).SetLocation(f).SetExpression(5).SetExpiration(time.Now()).MustBuild()

	envContext := func(ctx context.Context) context.Context {
		return context.WithValue(ctx, envMarkerKey("marker"), "stamped")
	}

	var gotMarker any
	processExpired(l, context.Background(), []Model{m}, func(_ logrus.FieldLogger, ctx context.Context, _ Model) error {
		gotMarker = ctx.Value(envMarkerKey("marker"))
		return nil
	}, envContext)

	if gotMarker != "stamped" {
		t.Fatalf("envContext was not applied to the revert context: got %v, want \"stamped\"", gotMarker)
	}
}
