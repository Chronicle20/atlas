package character

import (
	"atlas-parties/rest"
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
	return rest.MakeGetRequest[ForeignRestModel](fmt.Sprintf(getBaseRequest()+ById, id))
}
