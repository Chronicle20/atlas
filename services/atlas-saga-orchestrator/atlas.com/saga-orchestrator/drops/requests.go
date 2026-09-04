package drops

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	clearDropsPath = "worlds/%d/channels/%d/maps/%d/instances/%s/drops"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "DROPS")
}

func requestClearDrops(ctx context.Context, f field.Model) requests.EmptyBodyRequest {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return func(l logrus.FieldLogger, ctx context.Context) error { return err }
	}
	return requests.DeleteRequest(fmt.Sprintf(root+clearDropsPath, f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String()))
}
