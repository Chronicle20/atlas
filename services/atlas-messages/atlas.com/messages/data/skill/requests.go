package skill

import (
	"context"
	"fmt"
	"net/url"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	skillsResource       = "data/skills/%d"
	skillsSearchResource = "data/skills?name=%s"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "DATA")
}

func requestById(ctx context.Context, skillId uint32) requests.Request[RestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return requests.GetRequest[RestModel](fmt.Sprintf(root+skillsResource, skillId))
}

func requestByName(ctx context.Context, name string) requests.Request[[]RestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[[]RestModel](err)
	}
	return requests.GetRequest[[]RestModel](fmt.Sprintf(root+skillsSearchResource, url.QueryEscape(name)))
}
