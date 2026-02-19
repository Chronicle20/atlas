package guild

import (
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	Resource   = "guilds"
	ByMemberId = Resource + "?filter[members.id]=%d"
)

func getBaseRequest() string {
	return requests.RootUrl("GUILDS")
}

func requestByMemberId(id uint32) requests.Request[[]RestModel] {
	return requests.GetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+ByMemberId, id))
}
