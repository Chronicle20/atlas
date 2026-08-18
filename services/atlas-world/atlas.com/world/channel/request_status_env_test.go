package channel_test

import (
	"atlas-world/channel"
	tenant2 "atlas-world/configuration/tenant"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// requestStatusEnvMarkerKey is a test-local context key standing in for
// atlas-env's real marker. It pins that RequestStatus applies the injected
// envContext to each tenant's context before RequestStatusAndEmit's Kafka
// emit -- without the channel package importing atlas-env itself
// (env-domain-guard forbids that; main.go threads the real
// env.WithContext/env.Self() implementation in as a plain function value
// instead, task-232).
type requestStatusEnvMarkerKey struct{}

func TestRequestStatus_AppliesEnvContext(t *testing.T) {
	setupTestRegistry(t)
	tenantId := uuid.New()

	logger, _ := logtest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	var envContextCalled bool
	var seenTenantId string
	envContext := func(c context.Context) context.Context {
		envContextCalled = true
		seenTenant := tenant.MustFromContext(c)
		seenTenantId = seenTenant.Id().String()
		return context.WithValue(c, requestStatusEnvMarkerKey{}, "pod-env")
	}

	rm := tenant2.RestModel{
		Id:           tenantId.String(),
		Region:       "GMS",
		MajorVersion: 83,
		MinorVersion: 1,
	}

	err := channel.RequestStatus(logger, envContext)(context.Background())(tenantId)(rm)
	require.NoError(t, err)

	assert.True(t, envContextCalled, "RequestStatus must apply envContext to the per-tenant context before emitting")
	assert.Equal(t, tenantId.String(), seenTenantId, "envContext must be applied after tenant.WithContext, so it observes the swept tenant's own id")
}
