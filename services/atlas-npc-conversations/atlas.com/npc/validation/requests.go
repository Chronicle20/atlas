package validation

import (
	"atlas-npc-conversations/rest"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
)

func getBaseRequest() string {
	return requests.RootUrl("QUERY_AGGREGATOR")
}

func requestById(id uint32, body RestModel) requests.Request[RestModel] {
	return rest.MakePostRequest[RestModel](fmt.Sprint(getBaseRequest()+"/api/validations"), body)
}
