package workers

import (
	"atlas-data/canonical"
	"testing"

	"github.com/google/uuid"
)

// TestTenantFromParamsSharedVersionScoped verifies that scope=shared uses the
// version-derived canonical id (TenantId) rather than the all-zeros sentinel
// (TenantUUID). A shared ingest for GMS 84.1 must write to a different
// canonical partition than GMS 83.1.
func TestTenantFromParamsSharedVersionScoped(t *testing.T) {
	p84 := Params{ScopeKey: "shared", Region: "GMS", MajorVersion: 84, MinorVersion: 1}
	m84, err := tenantFromParams(p84)
	if err != nil {
		t.Fatalf("tenantFromParams GMS 84.1: %v", err)
	}

	want84 := canonical.TenantId("GMS", 84, 1)
	if m84.Id() != want84 {
		t.Fatalf("GMS 84.1 shared id = %s, want canonical TenantId = %s", m84.Id(), want84)
	}

	// Must NOT equal the all-zeros sentinel.
	zeroUUID := uuid.Nil
	if m84.Id() == zeroUUID {
		t.Fatalf("GMS 84.1 shared id is uuid.Nil — expected a non-zero version-derived uuid")
	}

	// The old behavior would have returned TenantUUID (all zeros); reject that.
	if m84.Id().String() == canonical.TenantUUID {
		t.Fatalf("GMS 84.1 shared id equals the legacy TenantUUID sentinel — version-scoped id required")
	}
}

// TestTenantFromParamsSharedDifferentVersionsDistinct verifies that two
// different versions produce distinct canonical ids.
func TestTenantFromParamsSharedDifferentVersionsDistinct(t *testing.T) {
	p83 := Params{ScopeKey: "shared", Region: "GMS", MajorVersion: 83, MinorVersion: 1}
	p84 := Params{ScopeKey: "shared", Region: "GMS", MajorVersion: 84, MinorVersion: 1}

	m83, err := tenantFromParams(p83)
	if err != nil {
		t.Fatalf("tenantFromParams GMS 83.1: %v", err)
	}
	m84, err := tenantFromParams(p84)
	if err != nil {
		t.Fatalf("tenantFromParams GMS 84.1: %v", err)
	}

	if m83.Id() == m84.Id() {
		t.Fatalf("GMS 83.1 and GMS 84.1 produced the same canonical id %s — version-scoped ids must be distinct", m83.Id())
	}
}

// TestTenantFromParamsTenantScopeUnchanged verifies the tenants/<uuid> branch
// still parses the suffix and returns that exact uuid (unchanged behavior).
func TestTenantFromParamsTenantScopeUnchanged(t *testing.T) {
	tenantId := "12345678-1234-1234-1234-123456789abc"
	p := Params{
		ScopeKey:     "tenants/" + tenantId,
		Region:       "GMS",
		MajorVersion: 83,
		MinorVersion: 1,
	}
	m, err := tenantFromParams(p)
	if err != nil {
		t.Fatalf("tenantFromParams tenants/ branch: %v", err)
	}
	if m.Id().String() != tenantId {
		t.Fatalf("tenants/ branch id = %s, want %s", m.Id(), tenantId)
	}
}

// TestTenantFromParamsInvalidScopeReturnsError verifies that an unrecognised
// scope key returns an error (unchanged default branch behavior).
func TestTenantFromParamsInvalidScopeReturnsError(t *testing.T) {
	p := Params{ScopeKey: "unknown/scope", Region: "GMS", MajorVersion: 83, MinorVersion: 1}
	_, err := tenantFromParams(p)
	if err == nil {
		t.Fatalf("expected error for invalid scope key, got nil")
	}
}
