package reactor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// envMarkerKey is a test-local context key. It stands in for atlas-env's
// real marker so this test can pin that DestroyInTenant threads its
// envContext parameter onto the tenant context before the teardown sweep's
// per-tenant destroy runs -- without the reactor package importing
// atlas-env itself (env-domain-guard forbids that; see the guard-compliance
// note on the Processor interface's Teardown/DestroyAll/DestroyInTenant
// methods).
type envMarkerKey struct{}

// TestDestroyInTenant_AppliesEnvContext pins that DestroyInTenant calls the
// injected envContext on the per-tenant context before handing it to the
// nested processor that emits the destroyed-status event -- the shutdown
// sweep iterates every tenant across every environment, so without this the
// emitted event would carry no ENVIRONMENT header and, per RootUrlFor's
// fail-open default, other pods would decode the event as if it were their
// own (FR-1.8).
func TestDestroyInTenant_AppliesEnvContext(t *testing.T) {
	tn := setupTestTenant()
	l := setupTestLogger()

	var capturedCtx context.Context
	envContext := func(ctx context.Context) context.Context {
		capturedCtx = ctx
		return context.WithValue(ctx, envMarkerKey{}, "pod-env")
	}

	p := &ProcessorImpl{l: l, ctx: context.Background()}
	op := p.DestroyInTenant(envContext, tn)

	err := op(nil)
	assert.NoError(t, err)

	if assert.NotNil(t, capturedCtx, "envContext was never invoked") {
		got, terr := tenant.FromContext(capturedCtx)()
		assert.NoError(t, terr)
		assert.Equal(t, tn, got)
	}
}
