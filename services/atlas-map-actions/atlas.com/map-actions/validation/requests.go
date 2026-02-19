package validation

import (
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
)

func getBaseRequest() string {
	return requests.RootUrl("QUERY_AGGREGATOR")
}

func requestById(body RestModel) requests.Request[RestModel] {
	return requests.PostRequest[RestModel](fmt.Sprint(getBaseRequest()+"validations"), body)
}
