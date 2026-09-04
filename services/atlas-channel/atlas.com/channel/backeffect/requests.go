package backeffect

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	mapInstanceResource           = "worlds/%d/channels/%d/maps/%d/instances/%s"
	mapInstanceBackEffectResource = mapInstanceResource + "/backEffects"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "MAPS")
}

func requestBackEffectsInMap(ctx context.Context, f field.Model) requests.Request[[]RestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[[]RestModel](err)
	}
	return requests.GetRequest[[]RestModel](fmt.Sprintf(root+mapInstanceBackEffectResource, f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String()))
}
