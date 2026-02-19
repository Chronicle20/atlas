package party

import (
	"atlas-query-aggregator/rest"
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	Resource      = "parties"
	ByMemberId    = Resource + "?filter[members.id]=%d"
)

func getBaseRequest() string {
	return requests.RootUrl("PARTIES")
}

func requestByMemberId(memberId uint32) requests.Request[[]RestModel] {
	return rest.MakeGetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+ByMemberId, memberId))
}
