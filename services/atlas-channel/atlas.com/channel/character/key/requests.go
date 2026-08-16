package key

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource = "characters/%d/keys"
	ByKey    = Resource + "/%d"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "KEYS")
}

// characterKeysUrl returns the list URL for a character's key map. It is a
// bare URL (not a requests.Request) because the list is now paginated
// server-side (task-117) and consumed via requests.DrainProvider, which
// appends its own page[number]/page[size] query params per request.
func characterKeysUrl(ctx context.Context, characterId uint32) (string, error) {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(root+Resource, characterId), nil
}

func updateKey(ctx context.Context, characterId uint32, key int32, theType int8, action int32) requests.Request[RestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	i := RestModel{
		Key:    key,
		Type:   theType,
		Action: action,
	}
	return requests.PatchRequest[RestModel](fmt.Sprintf(root+ByKey, characterId, key), i)
}
