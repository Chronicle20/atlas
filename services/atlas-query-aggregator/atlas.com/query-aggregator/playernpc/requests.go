package playernpc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// Resource is the eligibility predicate atlas-player-npcs exposes
// (design §9.1) -- the single place FR-1.1's automatic deploy check and
// FR-6.1's conversation condition both evaluate through, so they cannot
// disagree. worldId is deliberately part of the query string rather than
// left to the endpoint's own default (which is 0): a caller in a non-zero
// world that omitted it would silently mis-scope the duplicate check.
const Resource = "player-npcs/eligibility?characterId=%d&mapId=%d&worldId=%d"

// httpClient mirrors pendingchange/requests.go's direct-net/http pattern:
// this endpoint returns a plain JSON body (not JSON:API), so
// requests.GetRequest (which unmarshals via jsonapi.Unmarshal) cannot be
// used to decode it.
var httpClient = &http.Client{Timeout: 15 * time.Second}

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "PLAYER_NPCS")
}

func eligibilityUrl(ctx context.Context, characterId uint32, mapId uint32, worldId byte) (string, error) {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(root+Resource, characterId, mapId, worldId), nil
}

// requestEligibility issues the GET and decodes the plain JSON body directly.
func requestEligibility(l logrus.FieldLogger, ctx context.Context, characterId uint32, mapId uint32, worldId byte) (EligibilityRestModel, error) {
	var result EligibilityRestModel

	url, err := eligibilityUrl(ctx, characterId, mapId, worldId)
	if err != nil {
		return result, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return result, err
	}
	requests.TenantHeaderDecorator(ctx)(req.Header)
	requests.SpanHeaderDecorator(ctx)(req.Header)

	l.Debugf("Issuing [%s] request to [%s].", http.MethodGet, req.URL)
	resp, err := httpClient.Do(req)
	if err != nil {
		l.WithError(err).Warnf("Failed calling [%s] on [%s].", http.MethodGet, url)
		return result, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		l.WithError(err).Warnf("Failed reading response from [%s] on [%s].", http.MethodGet, url)
		return result, err
	}

	if resp.StatusCode != http.StatusOK {
		return result, fmt.Errorf("eligibility endpoint returned status %d: %s", resp.StatusCode, string(body))
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return result, err
	}
	return result, nil
}
