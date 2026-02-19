package etc

import (
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	itemInformationResource = "data/etcs/"
	itemInformationById     = itemInformationResource + "%d"
)

func getBaseRequest() string {
	return requests.RootUrl("DATA")
}

func requestById(id uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+itemInformationById, id))
}
