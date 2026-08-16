package saga

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// SagaById is atlas-saga-orchestrator's GET /sagas/{transactionId}
// (saga/resource.go:22).
const SagaById = "sagas/%s"

// getBaseRequest resolves the orchestrator's root for the environment on
// ctx. RootUrlFor falls back to BASE_SERVICE_URL when no SAGAS_SERVICE_URL
// is set, and the shared ingress already routes /api/sagas to
// atlas-saga-orchestrator (deploy/shared/routes.conf), so this needs no new
// deployment wiring — the same arrangement every other REST client in this
// service uses.
func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "SAGAS")
}

func requestSagaById(ctx context.Context, transactionId uuid.UUID) requests.Request[RestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return requests.GetRequest[RestModel](fmt.Sprintf(root+SagaById, transactionId.String()))
}
