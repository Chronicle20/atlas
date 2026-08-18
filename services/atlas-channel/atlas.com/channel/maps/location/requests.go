package location

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	characterconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
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
	State     string     `json:"state"`
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

// Model is the state-bearing projection of atlas-maps's character location.
// field.Model has nowhere to carry the presence discriminator, so /find reads
// this instead; GetField is unchanged for its existing callers.
type Model struct {
	characterId uint32
	worldId     world.Id
	channelId   channel.Id
	mapId       _map.Id
	instance    uuid.UUID
	state       characterconst.PresenceState
}

func (m Model) CharacterId() uint32                 { return m.characterId }
func (m Model) WorldId() world.Id                   { return m.worldId }
func (m Model) ChannelId() channel.Id               { return m.channelId }
func (m Model) MapId() _map.Id                      { return m.mapId }
func (m Model) Instance() uuid.UUID                 { return m.instance }
func (m Model) State() characterconst.PresenceState { return m.state }

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

// NewModelForTest constructs a Model directly. Only call from a test;
// production code builds one through Get.
func NewModelForTest(characterId uint32, w world.Id, ch channel.Id, m _map.Id, instance uuid.UUID, state characterconst.PresenceState) Model {
	return Model{characterId: characterId, worldId: w, channelId: ch, mapId: m, instance: instance, state: state}
}

// SetBaseURLForTest swaps the base URL for tests using httptest. Only
// call from a test; production code uses the env-driven default.
func SetBaseURLForTest(url string) func() {
	prev := baseURLProvider
	baseURLProvider = func(_ context.Context) (string, error) { return url + "/api/", nil }
	return func() { baseURLProvider = prev }
}
