package monster

import (
	"atlas-party-quests/rest"
	"fmt"

	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/google/uuid"
)

const (
	MonstersInField = "worlds/%d/channels/%d/maps/%d/instances/%s/monsters"
)

func getBaseRequest() string {
	return requests.RootUrl("MONSTERS")
}

func requestDestroyInField(worldId world.Id, channelId channel.Id, mapId _map.Id, instance uuid.UUID) requests.EmptyBodyRequest {
	return rest.MakeDeleteRequest(fmt.Sprintf(getBaseRequest()+MonstersInField, worldId, channelId, mapId, instance.String()))
}
