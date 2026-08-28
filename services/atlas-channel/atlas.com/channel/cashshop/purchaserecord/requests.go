package purchaserecord

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource = "accounts/%d/purchaseRecords/%d"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "CASHSHOP")
}

func requestByAccountIdAndSerialNumber(ctx context.Context, accountId uint32, serialNumber uint32) requests.Request[RestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return requests.GetRequest[RestModel](fmt.Sprintf(root+Resource, accountId, serialNumber))
}
