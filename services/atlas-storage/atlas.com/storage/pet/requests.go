package pet

import (
	"atlas-storage/rest"
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	petById = "pets/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("PETS")
}

func requestById(id uint32) requests.Request[RestModel] {
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+petById, id))
}
