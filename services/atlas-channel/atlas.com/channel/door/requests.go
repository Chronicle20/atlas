package door

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	resourceById    = "doors/%s"
	resourceInField = "worlds/%d/channels/%d/maps/%d/instances/%s/doors"
	resourceByOwner = "characters/%d/doors"
)

func getBaseRequest() string {
	return requests.RootUrl("DOORS")
}

func requestById(id string) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+resourceById, id))
}

func requestInField(f field.Model) requests.Request[[]RestModel] {
	return requests.GetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+resourceInField, f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String()))
}

func requestByOwner(ownerCharacterId uint32) requests.Request[[]RestModel] {
	return requests.GetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+resourceByOwner, ownerCharacterId))
}
