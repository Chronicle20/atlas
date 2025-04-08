package equipable

import (
	"atlas-inventory/rest"
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	equipmentResource = "equipables"
	equipResource     = equipmentResource + "/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("EQUIPABLES")
}

func requestById(equipmentId uint32) requests.Request[RestModel] {
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+equipResource, equipmentId))
}
