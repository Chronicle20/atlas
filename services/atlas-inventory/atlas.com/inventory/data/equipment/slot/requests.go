package slot

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	itemInformationResource = "data/equipment/"
	itemInformationById     = itemInformationResource + "%d"
	slotsForEquipment       = itemInformationById + "/slots"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "DATA")
}

func requestEquipmentSlotDestination(ctx context.Context, id uint32) requests.Request[[]RestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[[]RestModel](err)
	}
	return requests.GetRequest[[]RestModel](fmt.Sprintf(root+slotsForEquipment, id))
}
