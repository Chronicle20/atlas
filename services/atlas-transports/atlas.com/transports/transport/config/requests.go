package config

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	configurationsResource = "configurations"
	routesResource         = "routes"
	vesselsResource        = "vessels"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "TENANTS")
}

// routesUrl and vesselsUrl are bare URLs (not requests.Request) because
// both lists are now paginated server-side (task-117) and consumed via
// requests.DrainProvider, which appends its own page[number]/page[size]
// query params per request.
func routesUrl(ctx context.Context, tenantId string) (string, error) {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%stenants/%s/%s/%s", root, tenantId, configurationsResource, routesResource), nil
}

func vesselsUrl(ctx context.Context, tenantId string) (string, error) {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%stenants/%s/%s/%s", root, tenantId, configurationsResource, vesselsResource), nil
}
