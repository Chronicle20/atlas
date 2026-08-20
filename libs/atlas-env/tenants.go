package env

import (
	"context"
	"errors"
	"fmt"
)

// ctxKeyMismatch is a private context-key type so the mismatch flag can
// never collide with another package's context value, mirroring ctxKeyEnv
// in env.go.
type ctxKeyMismatchType string

const ctxKeyMismatch ctxKeyMismatchType = "env-mismatch"

// WithMismatch records that the operation's ENVIRONMENT header disagreed
// with the tenant it names (FR-7.7). A HeaderParser cannot return an error,
// so this is how the disagreement survives to the ownership gate, which
// drops the message.
func WithMismatch(ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxKeyMismatch, true)
}

// Mismatched reports whether WithMismatch was called on ctx (or an ancestor
// of it).
func Mismatched(ctx context.Context) bool {
	v, _ := ctx.Value(ctxKeyMismatch).(bool)
	return v
}

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
//
// CAUTION: the tenantId == "" short-circuit above is indistinguishable from
// an ordering bug that left the tenant off the context before Reconcile was
// called. A legitimately tenant-less message (no tenant on ctx yet) and a
// caller that forgot to register TenantHeaderParser before EnvHeaderParser
// both take this same path and both trust the header unconditionally. If
// messages start bypassing reconciliation unexpectedly, check parser
// registration order first.
func Reconcile(r Registry, headerEnv Id, tenantId string) (Id, error) {
	tr, ok := r.(TenantResolver)
	if !ok || tenantId == "" {
		return headerEnv, nil
	}
	tenantEnv, known := tr.EnvironmentOfTenant(tenantId)
	if !known {
		return headerEnv, nil
	}
	// A tenant projected with an EMPTY environment is LEGACY, not
	// "definitely belongs to no environment": a pre-#1427 tenant-status
	// event carried no environment attribute and MapRegistry.ApplyTenant
	// stores unconditionally (registry.go:67). Everywhere else in this
	// codebase "" means legacy-don't-filter (FR-1.8); treating it as a hard
	// mismatch here was the asymmetry that dropped every message a sparse
	// environment produced against a legacy tenant (FR-3.1).
	if tenantEnv == "" {
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
