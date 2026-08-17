package monster

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	mapMonstersResource = "worlds/%d/channels/%d/maps/%d/monsters"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "MONSTERS")
}

func requestCreate(ctx context.Context, f field.Model, monsterId uint32, x int16, y int16, fh int16, team int8) requests.Request[RestModel] {
	m := RestModel{
		Id:        "0",
		MonsterId: monsterId,
		X:         x,
		Y:         y,
		Fh:        fh,
		Team:      team,
	}
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	// TODO - field migration
	return requests.PostRequest[RestModel](fmt.Sprintf(root+mapMonstersResource, f.WorldId(), f.ChannelId(), f.MapId()), m)
}
