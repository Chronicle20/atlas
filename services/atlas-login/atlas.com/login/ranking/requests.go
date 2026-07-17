package ranking

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

const (
	Resource = "rankings/characters"
	ByIds    = Resource + "?ids=%s"

	// requestTimeout bounds the login-path call — login latency must never
	// ride on atlas-rankings health. The character-select decoration fails
	// open to zero-valued ranks on any error, including a timeout, so this
	// is deliberately short: the lib's default GET timeout (10s) is far too
	// long to hold up a character list.
	requestTimeout = 2 * time.Second
)

func getBaseRequest() string {
	return requests.RootUrl("RANKINGS")
}

// requestByCharacterIds builds a Request for the atlas-rankings bulk
// endpoint (GET /rankings/characters?ids=...), applying the login-path
// 2-second timeout via requests.SetTimeout. requests.GetRequest takes no
// configurators, so the timeout has to go through a hand-rolled
// requests.Request closure over requests.MakeGetRequest, matching the
// per-call-timeout pattern already established for the header decorators.
func requestByCharacterIds(ids []uint32) requests.Request[[]RestModel] {
	strs := make([]string, len(ids))
	for i, id := range ids {
		strs[i] = strconv.FormatUint(uint64(id), 10)
	}
	url := fmt.Sprintf(getBaseRequest()+ByIds, strings.Join(strs, ","))
	return func(l logrus.FieldLogger, ctx context.Context) ([]RestModel, error) {
		sd := requests.AddHeaderDecorator(requests.SpanHeaderDecorator(ctx))
		td := requests.AddHeaderDecorator(requests.TenantHeaderDecorator(ctx))
		return requests.MakeGetRequest[[]RestModel](url, sd, td, requests.SetTimeout(requestTimeout))(l, ctx)
	}
}
