package consumable

import (
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	consumableById = "data/consumables/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("DATA")
}

func requestById(id uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+consumableById, id))
}
