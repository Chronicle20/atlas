package wzinput

import (
	"net/http"
	"net/http/httptest"
	"testing"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
)

func mockTenant(t *testing.T) tenant.Model {
	t.Helper()
	m, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestResolveScopeDefault(t *testing.T) {
	r := httptest.NewRequest(http.MethodPatch, "/api/data/wz", nil)
	s, err := ResolveScope(r, mockTenant(t))
	if err != nil {
		t.Fatal(err)
	}
	if s.IsShared {
		t.Fatal("expected tenant scope")
	}
}

func TestResolveScopeSharedRequiresOperator(t *testing.T) {
	r := httptest.NewRequest(http.MethodPatch, "/api/data/wz?scope=shared", nil)
	_, err := ResolveScope(r, mockTenant(t))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestResolveScopeSharedWithOperator(t *testing.T) {
	r := httptest.NewRequest(http.MethodPatch, "/api/data/wz?scope=shared", nil)
	r.Header.Set("X-Atlas-Operator", "1")
	s, err := ResolveScope(r, mockTenant(t))
	if err != nil {
		t.Fatal(err)
	}
	if !s.IsShared {
		t.Fatal("expected shared scope")
	}
}

func TestResolveScopeInvalid(t *testing.T) {
	r := httptest.NewRequest(http.MethodPatch, "/api/data/wz?scope=bogus", nil)
	_, err := ResolveScope(r, mockTenant(t))
	if err == nil {
		t.Fatal("expected error")
	}
}
