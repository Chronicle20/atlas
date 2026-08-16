package door

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	resourceById    = "doors/%s"
	resourceInField = "worlds/%d/channels/%d/maps/%d/instances/%s/doors"
	resourceByOwner = "characters/%d/doors"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "DOORS")
}

func requestById(ctx context.Context, id string) requests.Request[RestModel] {

	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return requests.GetRequest[RestModel](fmt.Sprintf(root+resourceById, id))
}

// inFieldUrl returns the list URL for the doors currently in one map
// instance. It is a bare URL (not a requests.Request) because the list is
// now paginated server-side (task-117) and consumed via
// requests.DrainProvider, which appends its own page[number]/page[size]
// query params per request.
func inFieldUrl(ctx context.Context, f field.Model) string  {

	root, err := getBaseRequest(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(root+resourceInField, f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String()), nil
}

// byOwnerUrl returns the list URL for a character's doors. Bare URL for the
// same reason as inFieldUrl.
func byOwnerUrl(ctx context.Context, ownerCharacterId uint32) string  {

	root, err := getBaseRequest(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(root+resourceByOwner, ownerCharacterId), nil
}
