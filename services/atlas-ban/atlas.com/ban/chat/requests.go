package chat

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	HistoryByCharacterIds = "chat/history?characterIds=%s"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "MESSAGES")
}

func requestHistory(ctx context.Context, characterIds []uint32) requests.Request[[]RestModel] {
	ids := make([]string, 0, len(characterIds))
	for _, id := range characterIds {
		ids = append(ids, strconv.FormatUint(uint64(id), 10))
	}
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[[]RestModel](err)
	}
	return requests.GetRequest[[]RestModel](fmt.Sprintf(root+HistoryByCharacterIds, strings.Join(ids, ",")))
}
