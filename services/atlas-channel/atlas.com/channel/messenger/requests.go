package messenger

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource        = "messengers"
	ByMemberId      = Resource + "?filter[members.id]=%d"
	ById            = Resource + "/%d"
	MembersResource = ById + "/members"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "MESSENGERS")
}

func requestById(ctx context.Context, id uint32) requests.Request[RestModel] {

	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return requests.GetRequest[RestModel](fmt.Sprintf(root+ById, id))
}

func requestByMemberId(ctx context.Context, id uint32) requests.Request[[]RestModel]  {

	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[[]RestModel](err)
	}
	return requests.GetRequest[[]RestModel](fmt.Sprintf(root+ByMemberId, id))
}

func requestMembers(ctx context.Context, id uint32) requests.Request[[]MemberRestModel]  {

	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[[]MemberRestModel](err)
	}
	return requests.GetRequest[[]MemberRestModel](fmt.Sprintf(root+MembersResource, id))
}
