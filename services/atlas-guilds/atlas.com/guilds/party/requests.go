package party

import (
	"atlas-guilds/rest"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
	"os"
)

const (
	Resource   = "parties"
	ByMemberId = Resource + "?filter[members.id]=%d"
	ById       = Resource + "/%d"
	Members    = ById + "/members"
)

func getBaseRequest() string {
	return os.Getenv("BASE_SERVICE_URL")
}

func requestById(id uint32) requests.Request[RestModel] {
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+ById, id))
}

func requestByMemberId(id uint32) requests.Request[[]RestModel] {
	return rest.MakeGetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+ByMemberId, id))
}

func requestMembers(id uint32) requests.Request[[]MemberRestModel] {
	return rest.MakeGetRequest[[]MemberRestModel](fmt.Sprintf(getBaseRequest()+Members, id))
}
