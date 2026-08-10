package character

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource = "characters"
	ById     = Resource + "/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("CHARACTERS")
}

func requestById(id character.Id) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+ById, id))
}
