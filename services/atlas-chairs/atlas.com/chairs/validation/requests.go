package validation

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	validationsPath = "validations"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "QUERY_AGGREGATOR")
}

func requestValidation(ctx context.Context, characterId uint32, conditions []ConditionInput) requests.Request[ResponseModel] {
	body := RequestModel{
		Id:         characterId,
		Conditions: conditions,
	}
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[ResponseModel](err)
	}
	return requests.PostRequest[ResponseModel](root+validationsPath, body)
}
