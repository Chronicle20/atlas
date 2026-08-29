package itemmake

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	itemMakesResource    = "data/item-makes"
	itemMakeByIdResource = itemMakesResource + "/%d"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "DATA")
}

func requestById(ctx context.Context, itemId item.Id) requests.Request[RestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return requests.GetRequest[RestModel](fmt.Sprintf(root+itemMakeByIdResource, itemId))
}

// allItemMakesUrl returns the list URL for the full recipe catalog. It is a
// bare URL (not a requests.Request) because atlas-data's GET
// /data/item-makes is paginated (services/atlas-data/atlas.com/data/itemmake/resource.go
// calls paginate.ParseParams with a default page size of 50), the same
// task-117 rollout that made /data/quests and the atlas-skills/atlas-quests
// character lists paginated. It is consumed via requests.DrainProvider,
// which appends its own page[number]/page[size] query params per request;
// a single GetRequest would silently truncate the catalog past the first
// page.
func allItemMakesUrl(ctx context.Context) (string, error) {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return "", err
	}
	return root + itemMakesResource, nil
}
