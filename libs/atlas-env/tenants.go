package env

import (
	"errors"
	"fmt"
)

// ErrEnvironmentMismatch is FR-7.7: a hard error, never a reconciliation. A
// request whose ENVIRONMENT header disagrees with the tenant it names is
// rejected outright rather than having either side silently win.
var ErrEnvironmentMismatch = errors.New("environment header disagrees with the tenant's environment")

// TenantResolver is the half of Registry that answers "which environment
// does this tenant belong to". Separate interface so Reconcile is testable
// against a fake without the four routing queries.
type TenantResolver interface {
	EnvironmentOfTenant(tenantId string) (Id, bool)
}

// Reconcile resolves the operation's environment from the header and the
// tenant, and returns ErrEnvironmentMismatch when they disagree (FR-7.7).
// An unknown tenant trusts the header: during activation the tenant and
// environment records arrive on different topics and therefore different
// partitions, so a tenant may be visible before or after its environment
// (design §7.3). This does not weaken D4 — an unknown ENVIRONMENT is still
// rejected by the ownership gate.
func Reconcile(r Registry, headerEnv Id, tenantId string) (Id, error) {
	tr, ok := r.(TenantResolver)
	if !ok || tenantId == "" {
		return headerEnv, nil
	}
	tenantEnv, known := tr.EnvironmentOfTenant(tenantId)
	if !known {
		return headerEnv, nil
	}
	if headerEnv == "" {
		return tenantEnv, nil
	}
	if headerEnv != tenantEnv {
		return "", fmt.Errorf("%w: header=%q tenant=%q(%s)",
			ErrEnvironmentMismatch, headerEnv, tenantEnv, tenantId)
	}
	return headerEnv, nil
}
