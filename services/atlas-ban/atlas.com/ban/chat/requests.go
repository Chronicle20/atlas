package chat

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	HistoryByCharacterIds = "chat/history?characterIds=%s"
)

func getBaseRequest() string {
	return requests.RootUrl("MESSAGES")
}

func requestHistory(characterIds []uint32) requests.Request[[]RestModel] {
	ids := make([]string, 0, len(characterIds))
	for _, id := range characterIds {
		ids = append(ids, strconv.FormatUint(uint64(id), 10))
	}
	return requests.GetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+HistoryByCharacterIds, strings.Join(ids, ",")))
}
