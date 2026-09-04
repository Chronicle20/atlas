package broadcast_test

import (
	"atlas-world/broadcast"
	"atlas-world/test"
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// sweepEnvMarkerKey is a test-local context key standing in for atlas-env's
// real marker. It pins that Sweep.Run applies the injected envContext to
// each swept tenant's context before SweepTenant's Kafka emit -- without the
// broadcast package importing atlas-env itself (env-domain-guard forbids
// that; main.go threads the real env.WithContext/env.Self() implementation
// in as a plain function value instead, task-232).
type sweepEnvMarkerKey struct{}

func TestSweep_Run_AppliesEnvContext(t *testing.T) {
	setupTestRegistry(t)
	tenantId := uuid.New()
	ctx := test.CreateTestContextWithTenant(tenantId)
	logger, _ := logtest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	// Register the tenant in the broadcast registry via a real Enqueue so
	// GetRegistry().Tenants() (which Sweep.Run drives) returns it.
	processor := broadcast.NewProcessor(logger, ctx)
	require.NoError(t, processor.Enqueue(world.Id(1), broadcast.FamilyAvatar, broadcast.Entry{
		Id:              uuid.New(),
		CharacterId:     123,
		DurationSeconds: 30,
		Payload:         broadcast.Payload{SenderName: "Sender"},
	}))

	var envContextCalled bool
	var seenTenantId string
	envContext := func(c context.Context) context.Context {
		envContextCalled = true
		seenTenant := tenant.MustFromContext(c)
		seenTenantId = seenTenant.Id().String()
		return context.WithValue(c, sweepEnvMarkerKey{}, "pod-env")
	}

	sweep := broadcast.NewSweep(logger, time.Second, envContext)
	sweep.Run(ctx)

	assert.True(t, envContextCalled, "Sweep.Run must apply envContext to each swept tenant's context before emitting")
	assert.Equal(t, tenantId.String(), seenTenantId, "envContext must be applied after tenant.WithContext, so it observes the swept tenant's own id")
}
