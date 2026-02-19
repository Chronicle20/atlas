package mobskill

import (
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	mobSkillResource = "data/mob-skills/%d/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("DATA")
}

func requestByIdAndLevel(skillId uint16, level uint16) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+mobSkillResource, skillId, level))
}
