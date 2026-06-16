package summon

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	summonsInMapResource = "worlds/%d/channels/%d/maps/%d/instances/%s/summons"
)

func getBaseRequest() string {
	return requests.RootUrl("SUMMONS")
}

func requestInMap(f field.Model) requests.Request[[]RestModel] {
	return requests.GetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+summonsInMapResource, f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String()))
}
