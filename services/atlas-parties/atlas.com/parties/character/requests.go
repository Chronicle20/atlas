package character

import (
	"atlas-parties/rest"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
	"os"
)

const (
	Resource = "characters"
	ById     = Resource + "/%d"
)

func getBaseRequest() string {
	return os.Getenv("CHARACTER_SERVICE_URL")
}

func requestById(id uint32) requests.Request[ForeignRestModel] {
	return rest.MakeGetRequest[ForeignRestModel](fmt.Sprintf(getBaseRequest()+ById, id))
}
