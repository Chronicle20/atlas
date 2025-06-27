package skill

import (
	"atlas-character/rest"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	skillsResource = "data/skills/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("DATA")
}

func requestById(skillId uint32) requests.Request[RestModel] {
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+skillsResource, skillId))
}
