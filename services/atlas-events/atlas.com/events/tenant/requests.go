package tenant

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	tenantsResource = "tenants"
)

// getBaseRequest resolves the ingress for the environment carried on ctx
// rather than the process-wide baseline (task-232 FR-3.5/G4) — see the
// external/transports sibling for why the context-free requests.RootUrl
// would send this call into main from an ephemeral-environment pod.
func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "TENANTS")
}

// allTenantsUrl is a bare URL (not a requests.Request) because the list is
// now paginated server-side (task-117) and consumed via
// requests.DrainProvider, which appends its own page[number]/page[size]
// query params per request.
func allTenantsUrl(ctx context.Context) (string, error) {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return "", err
	}
	return root + tenantsResource, nil
}
