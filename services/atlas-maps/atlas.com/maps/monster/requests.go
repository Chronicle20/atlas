package monster

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	mapMonstersResource = "worlds/%d/channels/%d/maps/%d/instances/%s/monsters"
)

func getBaseRequest() string {
	return requests.RootUrl("MONSTERS")
}

// inMapUrl returns the list URL for the monsters currently in one map
// instance. It is a bare URL (not a requests.Request) because the list is
// now paginated server-side (task-117) and consumed via
// requests.DrainProvider, which appends its own page[number]/page[size]
// query params per request.
func inMapUrl(field field.Model) string {
	return fmt.Sprintf(getBaseRequest()+mapMonstersResource, field.WorldId(), field.ChannelId(), field.MapId(), field.Instance())
}

func requestCreate(field field.Model, monsterId uint32, x int16, y int16, fh int16, team int8) requests.Request[RestModel] {
	m := RestModel{
		Id:        "0",
		MonsterId: monsterId,
		X:         x,
		Y:         y,
		Fh:        fh,
		Team:      team,
	}
	return requests.PostRequest[RestModel](fmt.Sprintf(getBaseRequest()+mapMonstersResource, field.WorldId(), field.ChannelId(), field.MapId(), field.Instance()), m)
}
