package chalkboard

import (
	"atlas-channel/rest"
	"fmt"
	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	Resource = "worlds/%d/channels/%d/maps/%d/chalkboards"
)

func getBaseRequest() string {
	return requests.RootUrl("CHALKBOARDS")
}

func requestInMap(f field.Model) requests.Request[[]RestModel] {
	return rest.MakeGetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+Resource, f.WorldId(), f.ChannelId(), f.MapId()))
}
