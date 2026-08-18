package information

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	monstersResource = "data/monsters"
	monsterResource  = monstersResource + "/%d"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "DATA")
}
