package etc

import (
	"atlas-storage/rest"
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	etcById = "data/etcs/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("DATA")
}

func requestById(id uint32) requests.Request[RestModel] {
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+etcById, id))
}
