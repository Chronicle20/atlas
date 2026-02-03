package character

import (
	"atlas-effective-stats/rest"
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

// RequestById returns a request to fetch a character by ID from atlas-character service
func RequestById(id uint32) requests.Request[RestModel] {
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+ById, id))
}
