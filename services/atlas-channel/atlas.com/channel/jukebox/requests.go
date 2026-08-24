package jukebox

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	mapInstanceResource        = "worlds/%d/channels/%d/maps/%d/instances/%s"
	mapInstanceJukeboxResource = mapInstanceResource + "/jukebox"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "MAPS")
}

func requestJukeboxInMap(ctx context.Context, f field.Model) requests.Request[RestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return requests.GetRequest[RestModel](fmt.Sprintf(root+mapInstanceJukeboxResource, f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String()))
}
