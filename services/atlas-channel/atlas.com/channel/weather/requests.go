package weather

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	mapInstanceResource        = "worlds/%d/channels/%d/maps/%d/instances/%s"
	mapInstanceWeatherResource = mapInstanceResource + "/weather"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "MAPS")
}

func requestWeatherInMap(ctx context.Context, f field.Model) requests.Request[RestModel] {

	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return requests.GetRequest[RestModel](fmt.Sprintf(root+mapInstanceWeatherResource, f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String()))
}
