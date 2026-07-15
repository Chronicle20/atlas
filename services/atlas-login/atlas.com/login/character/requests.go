package character

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource          = "characters"
	ByAccountAndWorld = Resource + "?accountId=%d&worldId=%d"
	ByName            = Resource + "?name=%s"
	ById              = Resource + "/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("CHARACTERS")
}

// byAccountAndWorldUrl is a bare URL (not a requests.Request) because the
// list is now paginated server-side (task-117) and consumed via
// requests.DrainProvider, which appends its own page[number]/page[size]
// query params per request.
func byAccountAndWorldUrl(accountId uint32, worldId world.Id) string {
	return fmt.Sprintf(getBaseRequest()+ByAccountAndWorld, accountId, worldId)
}

func requestByName(name string) requests.Request[[]RestModel] {
	return requests.GetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+ByName, name))
}

func requestById(id uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+ById, id))
}

func requestDelete(id uint32) requests.EmptyBodyRequest {
	return requests.DeleteRequest(fmt.Sprintf(getBaseRequest()+ById, id))
}
