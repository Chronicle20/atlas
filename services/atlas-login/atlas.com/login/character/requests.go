package character

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource          = "characters"
	ByAccountAndWorld = Resource + "?accountId=%d&worldId=%d"
	ByName            = Resource + "?name=%s"
	ById              = Resource + "/%d"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "CHARACTERS")
}

// byAccountAndWorldUrl is a bare URL (not a requests.Request) because the
// list is now paginated server-side (task-117) and consumed via
// requests.DrainProvider, which appends its own page[number]/page[size]
// query params per request.
func byAccountAndWorldUrl(ctx context.Context, accountId uint32, worldId world.Id) (string, error) {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(root+ByAccountAndWorld, accountId, worldId), nil
}

func requestByName(ctx context.Context, name string) requests.Request[[]RestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[[]RestModel](err)
	}
	return requests.GetRequest[[]RestModel](fmt.Sprintf(root+ByName, name))
}

func requestById(ctx context.Context, id uint32) requests.Request[RestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return requests.GetRequest[RestModel](fmt.Sprintf(root+ById, id))
}

func requestDelete(ctx context.Context, id uint32) requests.EmptyBodyRequest {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return func(_ logrus.FieldLogger, _ context.Context) error { return err }
	}
	return requests.DeleteRequest(fmt.Sprintf(root+ById, id))
}
