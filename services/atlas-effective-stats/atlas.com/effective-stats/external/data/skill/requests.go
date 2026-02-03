package skill

import (
	"atlas-effective-stats/rest"
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	Resource  = "data/skills"
	SkillById = Resource + "/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("DATA")
}

// RequestById returns a request to fetch skill data by ID from atlas-data service
func RequestById(skillId uint32) requests.Request[RestModel] {
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+SkillById, skillId))
}
