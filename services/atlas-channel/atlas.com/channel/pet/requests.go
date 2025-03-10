package pet

import (
	"atlas-channel/rest"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	ByOwnerResource = "characters/%d/pets"
)

func getBaseRequest() string {
	return requests.RootUrl("PETS")
}

func requestByOwnerId(ownerId uint32) requests.Request[[]RestModel] {
	return rest.MakeGetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+ByOwnerResource, ownerId))
}
