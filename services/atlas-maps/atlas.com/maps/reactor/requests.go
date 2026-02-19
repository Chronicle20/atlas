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

func requestInMap(field field.Model) requests.Request[[]RestModel] {
	return requests.GetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+Resource, field.WorldId(), field.ChannelId(), field.MapId(), field.Instance()))
}
