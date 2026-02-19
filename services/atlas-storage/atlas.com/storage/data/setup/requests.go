package setup

import (
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	setupById = "data/setups/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("DATA")
}

func requestById(id uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+setupById, id))
}
