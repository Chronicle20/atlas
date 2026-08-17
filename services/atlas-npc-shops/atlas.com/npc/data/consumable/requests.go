package consumable

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource     = "data/consumables"
	ById         = Resource + "/%d"
	Rechargeable = Resource + "?fields[consumables]=rechargeable,slotMax,unitPrice&filter[rechargeable]=true"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "DATA")
}

func requestById(ctx context.Context, id uint32) requests.Request[RestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return requests.GetRequest[RestModel](fmt.Sprintf(root+ById, id))
}

// rechargeableUrl is a bare URL (not a requests.Request) because the
// filter[rechargeable]=true list is now paginated server-side (task-117)
// and consumed via requests.DrainProvider, which appends its own
// page[number]/page[size] query params per request.
func rechargeableUrl(ctx context.Context) (string, error) {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return "", err
	}
	return root + Rechargeable, nil
}
