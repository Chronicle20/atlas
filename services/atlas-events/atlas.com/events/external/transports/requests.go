package transports

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	routesResource = "transports/routes/%s"
)

func getBaseRequest() string {
	return requests.RootUrl("TRANSPORTS")
}

func requestRoute(routeId uuid.UUID) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+routesResource, routeId.String()))
}
