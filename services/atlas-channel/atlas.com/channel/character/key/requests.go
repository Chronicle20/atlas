package key

import (
	"atlas-channel/rest"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
	"os"
)

const (
	Resource = "characters/%d/keys"
)

func getBaseRequest() string {
	return os.Getenv("BASE_SERVICE_URL")
}

func requestByCharacterId(characterId uint32) requests.Request[[]RestModel] {
	return rest.MakeGetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+Resource, characterId))
}
