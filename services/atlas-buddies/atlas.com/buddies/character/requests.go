package character

import (
	"atlas-buddies/rest"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	Resource          = "characters"
	ById              = Resource + "/%d"
	ByIdWithInventory = Resource + "/%d?include=inventory"
	ByName            = Resource + "?name=%s&include=inventory"
)

func getBaseRequest() string {
	return requests.RootUrl("CHARACTERS")
}

func requestById(id uint32) requests.Request[RestModel] {
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+ById, id))
}
