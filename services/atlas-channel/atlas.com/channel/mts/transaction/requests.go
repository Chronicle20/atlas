package transaction

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// Resource is the atlas-mts transaction-history read endpoint:
// GET /characters/{characterId}/mts/transactions. It mirrors atlas-mts's
// transaction.handleGetCharacterTransactions.
const Resource = "characters/%d/mts/transactions"

func getBaseRequest() string {
	return requests.RootUrl("MTS")
}

func requestByCharacter(characterId uint32) requests.Request[[]RestModel] {
	return requests.GetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+Resource, characterId))
}
