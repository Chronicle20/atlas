package transports

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	routesResource = "transports/routes/%s"
)

// getBaseRequest resolves the ingress for the environment carried on ctx,
// not the process-wide baseline (task-232 FR-3.5/G4). requests.RootUrl, the
// context-free form, always returns BASE_SERVICE_URL — from a pod serving an
// ephemeral environment that address is main's ingress, so the call would
// silently transition into the wrong deployment. RootUrlFor never falls back:
// an unresolvable environment fails before the request is made.
func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "TRANSPORTS")
}

func requestRoute(ctx context.Context, routeId uuid.UUID) requests.Request[RestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return requests.GetRequest[RestModel](fmt.Sprintf(root+routesResource, routeId.String()))
}
