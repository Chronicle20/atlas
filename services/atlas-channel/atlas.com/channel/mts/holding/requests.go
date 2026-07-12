package holding

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// Resource is the atlas-mts take-home holding read endpoint:
// GET /characters/{characterId}/mts/holding. It mirrors atlas-mts's
// holding.handleGetCharacterHoldings.
const Resource = "characters/%d/mts/holding"

func getBaseRequest() string {
	return requests.RootUrl("MTS")
}

func requestByCharacter(characterId uint32) requests.Request[[]RestModel] {
	return requests.GetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+Resource, characterId))
}
