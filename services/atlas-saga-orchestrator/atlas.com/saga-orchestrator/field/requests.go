package field

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "MAPS")
}

func requestResetField(ctx context.Context, f field.Model, difficulty int) requests.Request[struct{}] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[struct{}](err)
	}
	url := fmt.Sprintf(root+"worlds/%d/channels/%d/maps/%d/instances/%s/reset",
		f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String())
	return requests.PostRequest[struct{}](url, ResetFieldInputRestModel{Difficulty: difficulty})
}
