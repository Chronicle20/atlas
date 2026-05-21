package workers

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func TestRegisteredSize(t *testing.T) {
	if len(Registered) != 11 {
		t.Fatalf("registered = %d, want 11", len(Registered))
	}
}

func TestRegisteredUniqueArchives(t *testing.T) {
	seen := map[string]bool{}
	for _, w := range Registered {
		if seen[w.ArchiveName()] {
			t.Fatalf("duplicate archive: %s", w.ArchiveName())
		}
		seen[w.ArchiveName()] = true
	}
}

func TestRegisteredUniqueNames(t *testing.T) {
	seen := map[string]bool{}
	for _, w := range Registered {
		if seen[w.Name()] {
			t.Fatalf("duplicate name: %s", w.Name())
		}
		seen[w.Name()] = true
	}
}

// TestWithTenantPreInjection locks in the dispatcher contract that
// data.RunWorkers MUST establish before invoking any Worker.Run: the
// context passed to Run carries the tenant from Params, so downstream
// tenant.MustFromContext calls don't panic regardless of how (or whether)
// individual workers also call withTenant. This guards against a recurrence
// of the Commodity panic from f247e976f.
func TestWithTenantPreInjection(t *testing.T) {
	p := Params{
		ScopeKey:     "tenants/00000000-0000-0000-0000-000000000001",
		Region:       "GMS",
		MajorVersion: 83,
		MinorVersion: 1,
	}
	ctx, model, err := WithTenant(context.Background(), p)
	if err != nil {
		t.Fatalf("WithTenant: %v", err)
	}
	if model.Id().String() != "00000000-0000-0000-0000-000000000001" {
		t.Fatalf("tenant id = %s, want from ScopeKey", model.Id())
	}
	// Round-trip: the resulting ctx MUST satisfy tenant.MustFromContext
	// without panicking. If WithTenant ever stops injecting (or starts
	// injecting against a key MustFromContext doesn't read), this test
	// surfaces it immediately rather than letting workers crash in prod.
	defer func() {
		if r := recover(); r != nil {
			if isTenantPanic(r, "retrieve id from context") {
				t.Fatalf("WithTenant did not put tenant where MustFromContext reads it (panic: %v)", r)
			}
			panic(r)
		}
	}()
	got := tenant.MustFromContext(ctx)
	if got.Id() != model.Id() {
		t.Fatalf("round-trip tenant id mismatch: %s vs %s", got.Id(), model.Id())
	}
}

func isTenantPanic(r interface{}, substr string) bool {
	switch v := r.(type) {
	case string:
		return strings.Contains(v, substr)
	case error:
		return strings.Contains(v.Error(), substr)
	default:
		return strings.Contains(fmt.Sprintf("%v", r), substr)
	}
}
