package pet

import (
	"context"
	"testing"

	"github.com/google/uuid"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// envMarkerKey is a test-local context key -- deliberately not
// libs/atlas-env, since pet sits outside env-domain-guard's permitted
// import list (main.go, kafka/, rest/, socket/) and must not import
// atlas-env even from a test file.
type envMarkerKey string

// TestTimeoutOwnerTenantContextAppliesEnvContext pins the task-232 batch-6
// origination-audit fix: ownerTenantContext must run the per-owner tenant
// context through envContext before EvaluateHungerAndEmit sees it, so the
// periodic hunger sweep's Kafka event carries this pod's own environment
// identity rather than an empty one. Without this, decide() fails open per
// FR-1.8 and the hunger event would be actioned by every live deployment,
// not just the originating one.
func TestTimeoutOwnerTenantContextAppliesEnvContext(t *testing.T) {
	tn, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("build tenant: %v", err)
	}

	envContext := func(ctx context.Context) context.Context {
		return context.WithValue(ctx, envMarkerKey("marker"), "stamped")
	}

	timeout := &Timeout{envContext: envContext}
	tctx := timeout.ownerTenantContext(context.Background(), tn)

	if got := tctx.Value(envMarkerKey("marker")); got != "stamped" {
		t.Fatalf("envContext was not applied: got %v, want \"stamped\"", got)
	}
	if got := tenant.MustFromContext(tctx); got != tn {
		t.Fatalf("tenant not preserved: got %v, want %v", got, tn)
	}
}
