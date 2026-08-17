package skill

import (
	"context"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	kafkaProducer "github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// envMarkerKey is a test-local context key standing in for atlas-env's real
// marker. It pins that ExpireCooldowns applies the injected envContext to
// each expired cooldown's per-tenant context before the status event is
// emitted -- without the skill package importing atlas-env itself
// (env-domain-guard forbids that; main.go, via tasks.ExpirationTask, threads
// the real env.WithContext/env.Self() implementation in as a plain function
// value instead).
type envMarkerKey struct{}

// expireCooldownsNoopWriter discards every message written to it, so
// ExpireCooldowns's real producer.ProviderImpl emission does not dial a
// real Kafka broker during the test.
type expireCooldownsNoopWriter struct{ topic string }

func (w expireCooldownsNoopWriter) Topic() string { return w.topic }
func (w expireCooldownsNoopWriter) WriteMessages(_ context.Context, _ ...kafka.Message) error {
	return nil
}
func (w expireCooldownsNoopWriter) Close() error { return nil }

// setupExpireCooldownsCapturingProducer installs a no-op producer writer
// factory so ExpireCooldowns's real emission has somewhere safe to land.
func setupExpireCooldownsCapturingProducer(t *testing.T) {
	t.Helper()
	kafkaProducer.ResetInstance()
	kafkaProducer.GetManager(kafkaProducer.ConfigWriterFactory(func(topicName string) kafkaProducer.Writer {
		return expireCooldownsNoopWriter{topic: topicName}
	}))
}

func TestExpireCooldowns_AppliesEnvContext(t *testing.T) {
	setupCooldownRegistryTest(t)
	setupExpireCooldownsCapturingProducer(t)
	ten := setupCooldownTestTenant(t)
	ctx := cooldownTestCtx(ten)

	characterId := uint32(3001)
	skillId := uint32(2001001)

	// cooldown=0 sets the expiration to "now"; sleep briefly so
	// CooldownExpiresAt().Before(time.Now()) is unambiguously true.
	require.NoError(t, GetRegistry().Apply(ctx, characterId, skillId, 0))
	time.Sleep(5 * time.Millisecond)

	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)

	var envContextCalled bool
	var seenTenantId string
	envContext := func(c context.Context) context.Context {
		envContextCalled = true
		seenTenant := tenant.MustFromContext(c)
		seenTenantId = seenTenant.Id().String()
		return context.WithValue(c, envMarkerKey{}, "pod-env")
	}

	ExpireCooldowns(l, ctx, envContext)

	assert.True(t, envContextCalled, "ExpireCooldowns must apply envContext to the expired cooldown's context before emitting")
	assert.Equal(t, ten.Id().String(), seenTenantId, "envContext must be applied after tenant.WithContext, so it observes the swept cooldown's own tenant")
}
