package wishlist

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource = "characters/%d/cash-shop/wishlist"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "CASHSHOP")
}

// byCharacterIdUrl returns the list URL for a character's cash-shop
// wishlist. It is a bare URL (not a requests.Request) because the list is
// now paginated server-side (task-117) and consumed via
// requests.DrainProvider, which appends its own page[number]/page[size]
// query params per request.
func byCharacterIdUrl(ctx context.Context, characterId uint32) string {

	root, err := getBaseRequest(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(root+Resource, characterId), nil
}

func addForCharacterId(ctx context.Context, characterId uint32, serialNumber uint32) requests.Request[RestModel]  {

	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return requests.PostRequest[RestModel](fmt.Sprintf(root+Resource, characterId), i)
}

func clearForCharacterId(ctx context.Context, characterId uint32) requests.EmptyBodyRequest  {

	root, err := getBaseRequest(ctx)
	if err != nil {
		return func(l logrus.FieldLogger, _ context.Context) error { return err }
	}
	return requests.DeleteRequest(fmt.Sprintf(root+Resource, characterId))
}
