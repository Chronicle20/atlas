package minigame

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource = "worlds/%d/channels/%d/maps/%d/instances/%s/games"
)

func getBaseRequest() string {
	return requests.RootUrl("MINI_GAMES")
}

func requestInField(f field.Model) requests.Request[[]RestModel] {
	return requests.GetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+Resource, f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String()))
}
