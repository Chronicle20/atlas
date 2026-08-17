package saved_location

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const BaseUrl = "CHARACTER_URL"

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, BaseUrl)
}

func PutSavedLocation(l logrus.FieldLogger, ctx context.Context) func(characterId uint32, locationType string, mapId _map.Id, portalId uint32) (RestModel, error) {
	return func(characterId uint32, locationType string, mapId _map.Id, portalId uint32) (RestModel, error) {
		body := RestModel{
			MapId:    mapId,
			PortalId: portalId,
		}
		root, err := getBaseRequest(ctx)
		if err != nil {
			return RestModel{}, err
		}
		url := fmt.Sprintf("%scharacters/%d/locations/%s", root, characterId, locationType)
		return requests.PutRequest[RestModel](url, body)(l, ctx)
	}
}

func GetSavedLocation(l logrus.FieldLogger, ctx context.Context) func(characterId uint32, locationType string) (RestModel, error) {
	return func(characterId uint32, locationType string) (RestModel, error) {
		root, err := getBaseRequest(ctx)
		if err != nil {
			return RestModel{}, err
		}
		url := fmt.Sprintf("%scharacters/%d/locations/%s", root, characterId, locationType)
		return requests.GetRequest[RestModel](url)(l, ctx)
	}
}

func DeleteSavedLocation(l logrus.FieldLogger, ctx context.Context) func(characterId uint32, locationType string) error {
	return func(characterId uint32, locationType string) error {
		root, err := getBaseRequest(ctx)
		if err != nil {
			return err
		}
		url := fmt.Sprintf("%scharacters/%d/locations/%s", root, characterId, locationType)
		return requests.DeleteRequest(url)(l, ctx)
	}
}
