package storage

import (
	"atlas-saga-orchestrator/rest"
	"context"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

const (
	storageAssetsResource = "storage/accounts/%d/assets?worldId=%d"
)

func getBaseRequest() string {
	return requests.RootUrl("STORAGE")
}

// RequestAssets retrieves all assets from storage for an account and world
func RequestAssets(l logrus.FieldLogger, ctx context.Context) func(accountId uint32, worldId byte) ([]AssetRestModel, error) {
	return func(accountId uint32, worldId byte) ([]AssetRestModel, error) {
		url := fmt.Sprintf(getBaseRequest()+storageAssetsResource, accountId, worldId)
		return rest.MakeGetRequest[[]AssetRestModel](url)(l, ctx)
	}
}
