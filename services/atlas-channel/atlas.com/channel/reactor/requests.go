package reactor

import (
	"fmt"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	Resource = "worlds/%d/channels/%d/maps/%d/instances/%s/reactors"
)

func getBaseRequest() string {
	return requests.RootUrl("REACTORS")
}

func requestInMap(f field.Model) requests.Request[[]RestModel] {
	return requests.GetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+Resource, f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String()))
}
