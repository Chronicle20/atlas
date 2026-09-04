package area_info

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// getBaseRequest returns the atlas-character base URL.
func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "CHARACTER_URL")
}

// requestAreaInfo requests the stored area-info string for a character/area.
func requestAreaInfo(ctx context.Context, characterId uint32, area uint16) requests.Request[RestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return requests.GetRequest[RestModel](fmt.Sprintf("%scharacters/%d/area-info/%d", root, characterId, area))
}
