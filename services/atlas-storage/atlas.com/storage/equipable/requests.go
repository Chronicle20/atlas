package equipable

import (
	"atlas-storage/rest"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	equipableById = "equipables/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("EQUIPABLES")
}

func requestById(id uint32) requests.Request[RestModel] {
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+equipableById, id))
}
