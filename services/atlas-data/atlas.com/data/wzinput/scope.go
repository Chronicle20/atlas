package wzinput

import (
	"errors"
	"net/http"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// Scope is the upload destination prefix.
type Scope struct {
	Key      string // "shared" or "tenants/<tenantId>"
	IsShared bool
}

// ErrSharedRequiresOperator is returned when scope=shared is requested without operator credentials.
var ErrSharedRequiresOperator = errors.New("scope=shared requires X-Atlas-Operator")

// ResolveScope reads the ?scope= query parameter and validates operator credentials.
// Default (empty or "tenant") returns the per-tenant scope.
func ResolveScope(r *http.Request, t tenant.Model) (Scope, error) {
	q := r.URL.Query().Get("scope")
	if q == "" || q == "tenant" {
		return Scope{Key: "tenants/" + t.Id().String(), IsShared: false}, nil
	}
	if q != "shared" {
		return Scope{}, errors.New("invalid scope")
	}
	if r.Header.Get("X-Atlas-Operator") != "1" {
		return Scope{}, ErrSharedRequiresOperator
	}
	return Scope{Key: "shared", IsShared: true}, nil
}
