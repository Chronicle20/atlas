package drop

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// identityEnvContext is a no-op envContext for tests that don't care about
// environment origination -- it just returns ctx unchanged.
func identityEnvContext(ctx context.Context) context.Context { return ctx }

// envMarkerKey is a test-local context key -- deliberately not
// libs/atlas-env, since drop sits outside env-domain-guard's permitted
// import list (main.go, kafka/, rest/, socket/) and must not import
// atlas-env even from a test file.
type envMarkerKey string

func TestNewExpirationTask_CreatesTaskWithCorrectValues(t *testing.T) {
	logger, _ := test.NewNullLogger()
	interval := 5 * time.Second

	task := NewExpirationTask(logger, interval, identityEnvContext)

	if task == nil {
		t.Fatal("Expected task to be created")
	}
	if task.l == nil {
		t.Fatal("Expected logger to be set")
	}
	if task.interval != interval {
		t.Fatalf("Expected interval %v, got %v", interval, task.interval)
	}
}

func TestExpirationTask_SleepTime_ReturnsInterval(t *testing.T) {
	logger, _ := test.NewNullLogger()
	interval := 10 * time.Second

	task := NewExpirationTask(logger, interval, identityEnvContext)

	if task.SleepTime() != interval {
		t.Fatalf("Expected SleepTime %v, got %v", interval, task.SleepTime())
	}
}

func TestExpirationTask_SleepTime_DifferentIntervals(t *testing.T) {
	logger, _ := test.NewNullLogger()

	tests := []time.Duration{
		1 * time.Second,
		30 * time.Second,
		1 * time.Minute,
		5 * time.Minute,
	}

	for _, interval := range tests {
		task := NewExpirationTask(logger, interval, identityEnvContext)
		if task.SleepTime() != interval {
			t.Errorf("Expected SleepTime %v, got %v", interval, task.SleepTime())
		}
	}
}

func TestExpirationTaskName_Constant(t *testing.T) {
	if ExpirationTaskName != "drop_expiration_task" {
		t.Fatalf("Expected ExpirationTaskName 'drop_expiration_task', got '%s'", ExpirationTaskName)
	}
}

// TestExpirationTaskTenantContextAppliesEnvContext pins the task-232 batch-2
// origination-audit fix: tenantContext must run each expired drop's
// per-tenant context through envContext before ExpireAndEmit produces its
// Kafka event, so the sweep carries this pod's own environment identity
// rather than an empty one. Without this the ExpirationTask sweep would fail
// FR-1.8's decide() open, and the expiration would be actioned by every live
// deployment, not just the originating one.
func TestExpirationTaskTenantContextAppliesEnvContext(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("build tenant: %v", err)
	}

	envContext := func(ctx context.Context) context.Context {
		return context.WithValue(ctx, envMarkerKey("marker"), "stamped")
	}

	task := NewExpirationTask(logger, time.Second, envContext)
	tctx := task.tenantContext(context.Background(), ten)

	if got := tctx.Value(envMarkerKey("marker")); got != "stamped" {
		t.Fatalf("envContext was not applied: got %v, want \"stamped\"", got)
	}
	if got, err := tenant.FromContext(tctx)(); err != nil || got != ten {
		t.Fatalf("tenant not preserved: got %v, err %v, want %v", got, err, ten)
	}
}
