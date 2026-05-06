package location

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// ErrNotFound is returned by GetField when atlas-maps reports HTTP 404
// (the character has no stored location row yet — usually first login of
// a freshly created character). Callers should distinguish this from
// infrastructure errors (5xx, network), which are returned as-is.
var ErrNotFound = errors.New("location not found")

const (
	Resource = "characters/%d/location"
)

// RestModel mirrors the JSON:API shape returned by atlas-maps's
// GET /characters/{id}/location endpoint. Required no-op relationship
// stubs are implemented per the api2go contract (see libs/atlas-rest CLAUDE.md).
type RestModel struct {
	Id        uint32     `json:"-"`
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
}

func (r RestModel) GetName() string { return "character-locations" }

func (r RestModel) GetID() string { return strconv.FormatUint(uint64(r.Id), 10) }

func (r *RestModel) SetID(s string) error {
	v, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return err
	}
	r.Id = uint32(v)
	return nil
}

func (r *RestModel) SetToOneReferenceID(_, _ string) error            { return nil }
func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }

var baseURLProvider = func() string {
	return requests.RootUrl("MAPS")
}

func requestByCharacterId(characterId uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(baseURLProvider()+Resource, characterId))
}

// GetField returns the durable field stored in atlas-maps for the given
// character. Caller must pass a logger and a context with tenant.
//
// On HTTP 404 (no location row yet), returns ErrNotFound — callers should
// treat this as the expected first-login condition. On any other error
// (5xx, network, decode), returns the underlying error so callers can
// distinguish infrastructure failures from missing data.
func GetField(l logrus.FieldLogger, ctx context.Context, characterId uint32) (field.Model, error) {
	rm, err := requestByCharacterId(characterId)(l, ctx)
	if err != nil {
		if errors.Is(err, requests.ErrNotFound) {
			return field.Model{}, ErrNotFound
		}
		return field.Model{}, err
	}
	return field.NewBuilder(rm.WorldId, rm.ChannelId, rm.MapId).SetInstance(rm.Instance).Build(), nil
}

// SetBaseURLForTest swaps the base URL for tests using httptest. Only
// call from a test; production code uses the env-driven default.
func SetBaseURLForTest(url string) func() {
	prev := baseURLProvider
	baseURLProvider = func() string { return url + "/api/" }
	return func() { baseURLProvider = prev }
}
