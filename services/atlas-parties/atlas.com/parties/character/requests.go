package character

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource = "characters"
	ById     = Resource + "/%d"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "CHARACTERS")
}

func requestById(ctx context.Context, id uint32) requests.Request[ForeignRestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[ForeignRestModel](err)
	}
	return requests.GetRequest[ForeignRestModel](fmt.Sprintf(root+ById, id))
}
