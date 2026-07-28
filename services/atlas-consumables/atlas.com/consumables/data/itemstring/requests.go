package itemstring

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource = "data/item-strings"
	ById     = Resource + "/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("DATA")
}

func requestById(id uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+ById, id))
}
