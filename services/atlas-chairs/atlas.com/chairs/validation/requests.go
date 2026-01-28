package validation

import (
	"atlas-chairs/rest"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	validationsPath = "validations"
)

func getBaseRequest() string {
	return requests.RootUrl("QUERY_AGGREGATOR")
}

func requestValidation(characterId uint32, conditions []ConditionInput) requests.Request[ResponseModel] {
	body := RequestModel{
		Id:         characterId,
		Conditions: conditions,
	}
	return rest.MakePostRequest[ResponseModel](getBaseRequest()+validationsPath, body)
}
