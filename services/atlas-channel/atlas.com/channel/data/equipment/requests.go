package equipment

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	equipmentResource = "data/equipment/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("DATA")
}

func requestById(itemId uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+equipmentResource, itemId))
}
