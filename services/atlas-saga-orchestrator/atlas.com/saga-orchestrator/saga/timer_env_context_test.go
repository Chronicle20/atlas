package saga

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// envMarkerKey is a test-local marker type -- never import atlas-env from a
// domain test file (8776709b8). It only proves the envContext closure
// SetEnvContext wires in is actually invoked by the timer-fire callback,
// which is what main.go's withSelfEnvironment relies on in production.
type envMarkerKey struct{}

// TestTimerRegistry_FireAppliesEnvContext pins that Schedule's AfterFunc
// callback applies the registry's envContext to the reconstructed
// context.Background()-rooted context before calling handleSagaTimeout.
// RED: before SetEnvContext existed, Schedule's fire callback built
// tenant.WithContext(context.Background(), t) with no hook for an injected
// environment, so a caller-supplied marker had no way to reach the fired
// context and this assertion could never pass. GREEN: SetEnvContext now
// wraps that context before the saga-timeout walk runs.
func TestTimerRegistry_FireAppliesEnvContext(t *testing.T) {
	ResetCache()
	logger, _ := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	var applied atomic.Bool
	SagaTimers().SetEnvContext(func(ctx context.Context) context.Context {
		applied.Store(true)
		return context.WithValue(ctx, envMarkerKey{}, "marked")
	})
	t.Cleanup(func() { SagaTimers().SetEnvContext(nil) })

	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), te)

	s, _ := NewBuilder().SetSagaType(CharacterCreation).SetInitiatedBy("test").Build()
	_ = GetCache().Put(ctx, s)

	SagaTimers().Schedule(logger, te, s.TransactionId(), 20*time.Millisecond)

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if !SagaTimers().Has(s.TransactionId()) {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	assert.True(t, applied.Load(), "envContext should have been applied when the timer fired")
}
