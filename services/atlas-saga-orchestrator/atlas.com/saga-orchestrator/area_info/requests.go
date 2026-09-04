package area_info

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const BaseUrl = "CHARACTER_URL"

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, BaseUrl)
}

func PutAreaInfo(l logrus.FieldLogger, ctx context.Context) func(characterId uint32, area uint16, info string) (RestModel, error) {
	return func(characterId uint32, area uint16, info string) (RestModel, error) {
		body := RestModel{
			Info: info,
		}
		root, err := getBaseRequest(ctx)
		if err != nil {
			return RestModel{}, err
		}
		url := fmt.Sprintf("%scharacters/%d/area-info/%d", root, characterId, area)
		return requests.PutRequest[RestModel](url, body)(l, ctx)
	}
}

func GetAreaInfo(l logrus.FieldLogger, ctx context.Context) func(characterId uint32, area uint16) (RestModel, error) {
	return func(characterId uint32, area uint16) (RestModel, error) {
		root, err := getBaseRequest(ctx)
		if err != nil {
			return RestModel{}, err
		}
		url := fmt.Sprintf("%scharacters/%d/area-info/%d", root, characterId, area)
		return requests.GetRequest[RestModel](url)(l, ctx)
	}
}
