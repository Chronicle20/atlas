package statistics

import (
	"atlas-character/rest"
	"atlas-character/tenant"
	"context"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
	"os"
)

const (
	equipmentResource = "equipment"
	equipResource     = equipmentResource + "/%d"
)

func getBaseRequest() string {
	return os.Getenv("EQUIPABLE_SERVICE_URL")
}

func requestCreate(ctx context.Context, tenant tenant.Model) func(itemId uint32) requests.Request[RestModel] {
	return func(itemId uint32) requests.Request[RestModel] {
		input := &RestModel{
			ItemId: itemId,
		}
		return rest.MakePostRequest[RestModel](ctx, tenant)(fmt.Sprintf(getBaseRequest()+equipmentResource), input)
	}
}

func requestById(ctx context.Context, tenant tenant.Model) func(equipmentId uint32) requests.Request[RestModel] {
	return func(equipmentId uint32) requests.Request[RestModel] {
		return rest.MakeGetRequest[RestModel](ctx, tenant)(fmt.Sprintf(getBaseRequest()+equipResource, equipmentId))
	}
}

func deleteById(ctx context.Context, tenant tenant.Model) func(equipmentId uint32) requests.EmptyBodyRequest {
	return func(equipmentId uint32) requests.EmptyBodyRequest {
		return rest.MakeDeleteRequest(ctx, tenant)(fmt.Sprintf(getBaseRequest()+equipResource, equipmentId))
	}
}
