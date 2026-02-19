package saved_location

import (
	"context"
	"fmt"

	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

const BaseUrl = "CHARACTER_URL"

func getBaseRequest() string {
	return requests.RootUrl(BaseUrl)
}

func PutSavedLocation(l logrus.FieldLogger, ctx context.Context) func(characterId uint32, locationType string, mapId _map.Id, portalId uint32) (RestModel, error) {
	return func(characterId uint32, locationType string, mapId _map.Id, portalId uint32) (RestModel, error) {
		body := RestModel{
			MapId:    mapId,
			PortalId: portalId,
		}
		url := fmt.Sprintf("%scharacters/%d/locations/%s", getBaseRequest(), characterId, locationType)
		return requests.PutRequest[RestModel](url, body)(l, ctx)
	}
}

func GetSavedLocation(l logrus.FieldLogger, ctx context.Context) func(characterId uint32, locationType string) (RestModel, error) {
	return func(characterId uint32, locationType string) (RestModel, error) {
		url := fmt.Sprintf("%scharacters/%d/locations/%s", getBaseRequest(), characterId, locationType)
		return requests.GetRequest[RestModel](url)(l, ctx)
	}
}

func DeleteSavedLocation(l logrus.FieldLogger, ctx context.Context) func(characterId uint32, locationType string) error {
	return func(characterId uint32, locationType string) error {
		url := fmt.Sprintf("%scharacters/%d/locations/%s", getBaseRequest(), characterId, locationType)
		return requests.DeleteRequest(url)(l, ctx)
	}
}
