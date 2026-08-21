package service

import (
	"context"

	env "github.com/Chronicle20/atlas/libs/atlas-env"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// TenantEnvironment originates the environment for tenant-scoped background
// work: the environment that owns the tenant already on ctx, falling back
// to this pod's own (env.Self()) when ctx carries no tenant or the registry
// does not know it. Periodic sweeps have no request or message to inherit
// ENVIRONMENT from, and the environment of tenant-scoped work is a property
// of the TENANT, not of the pod that happens to run the sweep -- a baseline
// pod serving a sparse environment's tenant must stamp that environment,
// not its own. Without this, a background task's downstream event would be
// dropped by every consumer's ownership gate (FR-7.7) or actioned by every
// live deployment (FR-1.8), depending on which fallback would otherwise be
// used.
func TenantEnvironment(ctx context.Context) context.Context {
	self := env.Self()
	t, err := tenant.FromContext(ctx)()
	if err != nil {
		return env.WithContext(ctx, self)
	}
	return env.WithContext(ctx, env.ForTenant(env.CurrentRegistry(), t.Id().String(), self))
}
