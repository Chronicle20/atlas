package merchant

import (
	"fmt"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	Resource         = "worlds/%d/channels/%d/maps/%d/instances/%s/merchants"
	VisitingResource = "characters/%d/visiting"
)

func getBaseRequest() string {
	return requests.RootUrl("MERCHANT")
}

func requestInField(f field.Model) requests.Request[[]RestModel] {
	return requests.GetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+Resource, f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String()))
}

func requestVisiting(characterId uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+VisitingResource, characterId))
}
