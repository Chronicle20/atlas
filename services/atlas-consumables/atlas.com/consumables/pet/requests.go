package pet

import (
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	Resource        = "pets"
	ById            = Resource + "/%d"
	ByOwnerResource = "characters/%d/pets"
)

func getBaseRequest() string {
	return requests.RootUrl("PETS")
}

func requestById(petId uint64) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+ById, petId))
}

func requestByOwnerId(ownerId uint32) requests.Request[[]RestModel] {
	return requests.GetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+ByOwnerResource, ownerId))
}
