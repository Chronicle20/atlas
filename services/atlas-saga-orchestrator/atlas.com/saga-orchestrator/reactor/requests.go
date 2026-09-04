package reactor

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

func getReactorsBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "REACTORS")
}

func requestReactorsByName(ctx context.Context, worldId world.Id, channelId channel.Id, mapId _map.Id, instance uuid.UUID, name string) requests.Request[[]ReactorRestModel] {
	root, err := getReactorsBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[[]ReactorRestModel](err)
	}
	return requests.GetRequest[[]ReactorRestModel](fmt.Sprintf(
		root+"worlds/%d/channels/%d/maps/%d/instances/%s/reactors?name=%s",
		worldId, channelId, mapId, instance.String(), name,
	))
}

func requestResetReactors(ctx context.Context, f field.Model, minState *int8) requests.Request[struct{}] {
	root, err := getReactorsBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[struct{}](err)
	}
	url := fmt.Sprintf(root+"worlds/%d/channels/%d/maps/%d/instances/%s/reactors/reset",
		f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String())
	return requests.PostRequest[struct{}](url, ResetReactorsInputRestModel{MinState: minState})
}

func requestShuffleReactors(ctx context.Context, f field.Model) requests.Request[struct{}] {
	root, err := getReactorsBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[struct{}](err)
	}
	url := fmt.Sprintf(root+"worlds/%d/channels/%d/maps/%d/instances/%s/reactors/shuffle",
		f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String())
	return requests.PostRequest[struct{}](url, ShuffleReactorsInputRestModel{})
}
