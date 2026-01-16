package cash

import (
	"atlas-inventory/rest"
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

const (
	itemsResource = "cash-shop/items"
	itemResource  = itemsResource + "/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("CASHSHOP")
}

func requestById(id uint32) requests.Request[RestModel] {
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+itemResource, id))
}

func requestDelete(l logrus.FieldLogger, ctx context.Context) func(id uint32) error {
	return func(id uint32) error {
		url := fmt.Sprintf(getBaseRequest()+itemResource, id)
		return rest.MakeDeleteRequest(url)(l, ctx)
	}
}
