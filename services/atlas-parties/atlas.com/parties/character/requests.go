package character

import (
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	Resource = "characters"
	ById     = Resource + "/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("CHARACTERS")
}

func requestById(id uint32) requests.Request[ForeignRestModel] {
	return requests.GetRequest[ForeignRestModel](fmt.Sprintf(getBaseRequest()+ById, id))
}
