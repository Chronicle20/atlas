package character

import (
	"atlas-login/rest"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	Resource          = "characters"
	ByAccountAndWorld = Resource + "?accountId=%d&worldId=%d&include=inventory"
	ByName            = Resource + "?name=%s&include=inventory"
	ById              = Resource + "/%d?include=inventory"
)

func getBaseRequest() string {
	return requests.RootUrl("CHARACTERS")
}

func requestByAccountAndWorld(accountId uint32, worldId byte) requests.Request[[]RestModel] {
	return rest.MakeGetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+ByAccountAndWorld, accountId, worldId))
}

func requestByName(name string) requests.Request[[]RestModel] {
	return rest.MakeGetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+ByName, name))
}

func requestById(id uint32) requests.Request[RestModel] {
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+ById, id))
}

func requestDelete(id uint32) requests.EmptyBodyRequest {
	return rest.MakeDeleteRequest(fmt.Sprintf(getBaseRequest()+ById, id))
}
