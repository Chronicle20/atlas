package party

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource   = "parties"
	ByMemberId = Resource + "?filter[members.id]=%d"
	ById       = Resource + "/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("PARTIES")
}

func requestById(id uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+ById, id))
}

func requestByMemberId(id uint32) requests.Request[[]RestModel] {
	return requests.GetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+ByMemberId, id))
}
