package compartment

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource = "characters/%d/inventory/compartments"
	ByType   = Resource + "?type=%d"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "INVENTORY")
}

func requestByType(ctx context.Context, characterId uint32, inventoryType inventory.Type) requests.Request[RestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return requests.GetRequest[RestModel](fmt.Sprintf(root+ByType, characterId, inventoryType))
}
