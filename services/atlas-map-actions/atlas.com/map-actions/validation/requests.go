package validation

import (
	"atlas-map-actions/rest"
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
)

func getBaseRequest() string {
	return requests.RootUrl("QUERY_AGGREGATOR")
}

func requestById(body RestModel) requests.Request[RestModel] {
	return rest.MakePostRequest[RestModel](fmt.Sprint(getBaseRequest()+"validations"), body)
}
