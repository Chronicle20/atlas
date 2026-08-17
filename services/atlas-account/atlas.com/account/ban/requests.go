package ban

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	BansCheck = "bans/check?ip=%s&hwid=%s&accountId=%d"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "BANS")
}

func requestCheckBan(ctx context.Context, ip string, hwid string, accountId uint32) requests.Request[CheckRestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[CheckRestModel](err)
	}
	return requests.GetRequest[CheckRestModel](fmt.Sprintf(root+BansCheck, ip, hwid, accountId))
}
