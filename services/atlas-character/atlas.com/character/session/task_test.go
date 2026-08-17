package session

import (
	"context"
	"testing"

	"github.com/google/uuid"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// envMarkerKey is a test-local context key -- deliberately not
// libs/atlas-env, since session sits outside env-domain-guard's permitted
// import list (main.go, kafka/, rest/, socket/) and must not import
// atlas-env even from a test file.
type envMarkerKey string

// TestTimeoutSessionTenantContextAppliesEnvContext pins the task-232 batch-2
// origination-audit fix: sessionTenantContext must run the per-character
// tenant context through envContext before LogoutAndEmit/EndSession see it,
// so a timed-out session's logout carries this pod's own environment
// identity rather than an empty one. Without this, the Timeout sweep would
// fail FR-1.8's decide() open, and the logout Kafka event would be actioned
// by every live deployment, not just the originating one.
func TestTimeoutSessionTenantContextAppliesEnvContext(t *testing.T) {
	tn, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("build tenant: %v", err)
	}
	m := Model{tenant: tn}

	envContext := func(ctx context.Context) context.Context {
		return context.WithValue(ctx, envMarkerKey("marker"), "stamped")
	}

	timeout := &Timeout{envContext: envContext}
	tctx := timeout.sessionTenantContext(context.Background(), m)

	if got := tctx.Value(envMarkerKey("marker")); got != "stamped" {
		t.Fatalf("envContext was not applied: got %v, want \"stamped\"", got)
	}
	if got := tenant.MustFromContext(tctx); got != tn {
		t.Fatalf("tenant not preserved: got %v, want %v", got, tn)
	}
}
