package pet

import (
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	Resource = "data/pets"
	ById     = Resource + "/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("DATA")
}

func requestById(petId uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+ById, petId))
}
