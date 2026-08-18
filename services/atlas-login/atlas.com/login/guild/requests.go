package guild

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource   = "guilds"
	ByMemberId = Resource + "?filter[members.id]=%d"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "GUILDS")
}

func requestByMemberId(ctx context.Context, id uint32) requests.Request[[]RestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[[]RestModel](err)
	}
	return requests.GetRequest[[]RestModel](fmt.Sprintf(root+ByMemberId, id))
}
