package weather

import (
	"atlas-channel/rest"
	"fmt"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	mapInstanceResource        = "worlds/%d/channels/%d/maps/%d/instances/%s"
	mapInstanceWeatherResource = mapInstanceResource + "/weather"
)

func getBaseRequest() string {
	return requests.RootUrl("MAPS")
}

func requestWeatherInMap(f field.Model) requests.Request[RestModel] {
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+mapInstanceWeatherResource, f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String()))
}
