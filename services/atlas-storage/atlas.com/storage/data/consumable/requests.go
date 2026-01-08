package consumable

import (
	"atlas-storage/rest"
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
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+consumableById, id))
}
