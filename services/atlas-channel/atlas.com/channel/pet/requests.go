package pet

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource        = "pets"
	ById            = Resource + "/%d"
	ByOwnerResource = "characters/%d/pets"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "PETS")
}

func requestById(ctx context.Context, petId uint32) requests.Request[RestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return requests.GetRequest[RestModel](fmt.Sprintf(root+ById, petId))
}

// byOwnerUrl returns the list URL for a character's pets. It is a bare URL
// (not a requests.Request) because the list is now paginated server-side
// (task-117) and consumed via requests.DrainProvider, which appends its own
// page[number]/page[size] query params per request.
func byOwnerUrl(ctx context.Context, ownerId uint32) (string, error) {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(root+ByOwnerResource, ownerId), nil
}
