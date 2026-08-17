package character

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource          = "characters"
	ById              = Resource + "/%d"
	ByIdWithInventory = Resource + "/%d?include=inventory"
	ByName            = Resource + "?name=%s&include=inventory"
	ByAccountInWorld  = Resource + "?accountId=%d&worldId=%d"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "CHARACTERS")
}

func requestById(ctx context.Context, id uint32) requests.Request[RestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return requests.GetRequest[RestModel](fmt.Sprintf(root+ById, id))
}

func requestByName(ctx context.Context, name string) requests.Request[[]RestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[[]RestModel](err)
	}
	return requests.GetRequest[[]RestModel](fmt.Sprintf(root+ByName, name))
}

// accountInWorldUrl returns the list URL for an account's characters in a
// world (atlas-character resource.go:33-34,99,
// "get_characters_for_account_in_world"). It is a bare URL (not a
// requests.Request), because that endpoint is served by
// character.Processor.GetForAccountInWorldProvider — a PAGED provider — so
// the response is a paged JSON:API document, not the bare array
// requestByName's []RestModel return type assumes. Callers drain it with
// requests.DrainProvider, mirroring note/requests.go's characterNotesUrl.
func accountInWorldUrl(ctx context.Context, accountId uint32, worldId world.Id) (string, error) {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(root+ByAccountInWorld, accountId, worldId), nil
}
