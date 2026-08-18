package location

import (
	"context"
	"errors"
	"fmt"

	"github.com/sirupsen/logrus"

	characterconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// ErrNotFound is returned by GetField when atlas-maps reports HTTP 404
// (the character has no stored location row yet — usually first login of
// a freshly created character). Callers should distinguish this from
// infrastructure errors (5xx, network), which are returned as-is.
var ErrNotFound = errors.New("location not found")

var baseURLProvider = func(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "MAPS")
}

func requestByCharacterId(ctx context.Context, characterId uint32) requests.Request[RestModel] {
	root, err := baseURLProvider(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return requests.GetRequest[RestModel](fmt.Sprintf(root+Resource, characterId))
}

// GetField returns the durable field stored in atlas-maps for the given
// character. Caller must pass a logger and a context with tenant.
//
// On HTTP 404 (no location row yet), returns ErrNotFound — callers should
// treat this as the expected first-login condition. On any other error
// (5xx, network, decode), returns the underlying error so callers can
// distinguish infrastructure failures from missing data.
func GetField(l logrus.FieldLogger, ctx context.Context, characterId uint32) (field.Model, error) {
	rm, err := requestByCharacterId(ctx, characterId)(l, ctx)
	if err != nil {
		if errors.Is(err, requests.ErrNotFound) {
			return field.Model{}, ErrNotFound
		}
		return field.Model{}, err
	}
	return field.NewBuilder(rm.WorldId, rm.ChannelId, rm.MapId).SetInstance(rm.Instance).Build(), nil
}

// Get returns the character's stored location including the presence state.
//
// On HTTP 404 (no location row at all — a character who has never logged in)
// it returns ErrNotFound. On any other error (5xx, network, decode) it returns
// the underlying error, so callers can distinguish infrastructure failure from
// missing data; /find logs those two at different levels.
//
// An absent or unrecognised state resolves to OFFLINE, so an atlas-maps that
// has not been redeployed degrades /find to "not findable" rather than to a
// fabricated channel.
func Get(l logrus.FieldLogger, ctx context.Context, characterId uint32) (Model, error) {
	rm, err := requestByCharacterId(ctx, characterId)(l, ctx)
	if err != nil {
		if errors.Is(err, requests.ErrNotFound) {
			return Model{}, ErrNotFound
		}
		return Model{}, err
	}
	return Model{
		characterId: characterId,
		worldId:     rm.WorldId,
		channelId:   rm.ChannelId,
		mapId:       rm.MapId,
		instance:    rm.Instance,
		state:       characterconst.ParsePresenceState(rm.State),
	}, nil
}

// SetBaseURLForTest swaps the base URL for tests using httptest. Only
// call from a test; production code uses the env-driven default.
func SetBaseURLForTest(url string) func() {
	prev := baseURLProvider
	baseURLProvider = func(_ context.Context) (string, error) { return url + "/api/", nil }
	return func() { baseURLProvider = prev }
}
