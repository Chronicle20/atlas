package pet

import (
	"atlas-pets/rest"
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
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+ById, petId))
}
