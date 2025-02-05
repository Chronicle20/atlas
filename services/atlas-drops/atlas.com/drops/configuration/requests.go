package configuration

import (
	"atlas-drops/rest"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/google/uuid"
	"os"
)

const (
	Resource  = "configurations"
	ByService = Resource + "/%s?id=%s"
)

func getBaseRequest() string {
	return os.Getenv("BASE_SERVICE_URL")
}

func requestByService(serviceId uuid.UUID, serviceType string) requests.Request[RestModel] {
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+ByService, serviceType, serviceId.String()))
}
