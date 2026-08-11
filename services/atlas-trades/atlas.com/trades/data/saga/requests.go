package saga

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// SagaById is atlas-saga-orchestrator's GET /sagas/{transactionId}
// (saga/resource.go:22).
const SagaById = "sagas/%s"

// getBaseRequest resolves the orchestrator's root. RootUrl falls back to
// BASE_SERVICE_URL when no SAGAS_SERVICE_URL is set, and the shared ingress
// already routes /api/sagas to atlas-saga-orchestrator
// (deploy/shared/routes.conf), so this needs no new deployment wiring — the
// same arrangement every other REST client in this service uses.
func getBaseRequest() string {
	return requests.RootUrl("SAGAS")
}

func requestSagaById(transactionId uuid.UUID) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+SagaById, transactionId.String()))
}
