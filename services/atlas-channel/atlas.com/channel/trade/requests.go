package trade

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	// RoomsResource is atlas-trades' read-only room list. The character filter
	// matches EITHER side of a room, so it answers occupancy for an owner and an
	// invitee alike
	// (services/atlas-trades/atlas.com/trades/trade/resource.go:110-127).
	RoomsResource = "trades/rooms?filter[characterId]=%d"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "TRADES")
}

// requestByMember reads the (0-or-1) trade room characterId occupies.
func requestByMember(ctx context.Context, characterId character.Id) requests.Request[[]RestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[[]RestModel](err)
	}
	return requests.GetRequest[[]RestModel](fmt.Sprintf(root+RoomsResource, characterId))
}
