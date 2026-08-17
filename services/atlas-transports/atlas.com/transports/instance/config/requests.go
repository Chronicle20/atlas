package config

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	configurationsResource = "configurations"
	instanceRoutesResource = "instance-routes"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "TENANTS")
}

// instanceRoutesUrl is a bare URL (not a requests.Request) because the list
// is now paginated server-side (task-117) and consumed via
// requests.DrainProvider, which appends its own page[number]/page[size]
// query params per request.
func instanceRoutesUrl(ctx context.Context, tenantId string) (string, error) {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%stenants/%s/%s/%s", root, tenantId, configurationsResource, instanceRoutesResource), nil
}
