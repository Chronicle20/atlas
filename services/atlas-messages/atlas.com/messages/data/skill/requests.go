package skill

import (
	"fmt"
	"net/url"

	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	skillsResource       = "data/skills/%d"
	skillsSearchResource = "data/skills?name=%s"
)

func getBaseRequest() string {
	return requests.RootUrl("DATA")
}

func requestById(skillId uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+skillsResource, skillId))
}

func requestByName(name string) requests.Request[[]RestModel] {
	return requests.GetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+skillsSearchResource, url.QueryEscape(name)))
}
