package guild

import (
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	Resource   = "guilds"
	ByMemberId = Resource + "?filter[members.id]=%d"
	ById       = Resource + "/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("GUILDS")
}

func requestById(id uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+ById, id))
}

func requestByMemberId(id uint32) requests.Request[[]RestModel] {
	return requests.GetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+ByMemberId, id))
}
