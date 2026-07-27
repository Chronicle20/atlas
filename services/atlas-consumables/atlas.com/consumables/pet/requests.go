package pet

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource        = "pets"
	ById            = Resource + "/%d"
	ByOwnerResource = "characters/%d/pets"
)

func getBaseRequest() string {
	return requests.RootUrl("PETS")
}

func requestById(petId uint64) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+ById, petId))
}

// byOwnerUrl returns the list URL for a character's pets. It is a bare URL
// (not a requests.Request) because the list is now paginated server-side
// (task-117) and consumed via requests.DrainProvider, which appends its own
// page[number]/page[size] query params per request.
func byOwnerUrl(ownerId uint32) string {
	return fmt.Sprintf(getBaseRequest()+ByOwnerResource, ownerId)
}
