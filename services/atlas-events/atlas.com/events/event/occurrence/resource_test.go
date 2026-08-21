package occurrence

import (
	"atlas-events/event/registry"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type testServerInformation struct{}

func (t *testServerInformation) GetBaseURL() string { return "http://localhost:8080" }
func (t *testServerInformation) GetPrefix() string  { return "/api/" }

var _ jsonapi.ServerInformation = &testServerInformation{}

func setupOccurrenceRouter(db *gorm.DB) *mux.Router {
	r := mux.NewRouter()
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	ri := InitResource(&testServerInformation{})(db)
	ri(r, l)
	return r
}

func requestWithTenant(method, url string, tenantId uuid.UUID) *http.Request {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("TENANT_ID", tenantId.String())
	req.Header.Set("REGION", "GMS")
	req.Header.Set("MAJOR_VERSION", "83")
	req.Header.Set("MINOR_VERSION", "1")
	return req
}

// doGET drives a request against db's tenant (testTenantId — see testCtx)
// through the real router and decodes the JSON:API "data" array's attributes
// into T. It is the harness the FR-B13/FR-B14/FR-API5/FR-API9 tests below
// share; see event/definition/resource_test.go for the sibling pattern this
// copies.
func doGET[T any](t *testing.T, db *gorm.DB, path string) []T {
	t.Helper()
	return doGETAsTenant[T](t, db, path, testTenantId)
}

func doGETAsTenant[T any](t *testing.T, db *gorm.DB, path string, tenantId uuid.UUID) []T {
	t.Helper()

	srv := httptest.NewServer(setupOccurrenceRouter(db))
	defer srv.Close()

	req := requestWithTenant(http.MethodGet, srv.URL+path, tenantId)
	resp, err := (&http.Client{}).Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	require.Equal(t, http.StatusOK, resp.StatusCode, "GET %s", path)

	var doc jsonapi.Document
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))

	items := make([]T, 0)
	if doc.Data == nil {
		return items
	}
	for _, d := range doc.Data.DataArray {
		var item T
		require.NoError(t, json.Unmarshal(d.Attributes, &item))
		items = append(items, item)
	}
	return items
}

// FR-B13/FR-B14: the cabin's map row has visual=false, so the projection
// returns nothing for it. This is what makes the deck/cabin distinction a
// query predicate rather than a branch in atlas-channel.
func TestVisualsProjectionExcludesNonVisualMaps(t *testing.T) {
	db := newTestDB(t)
	p := NewProcessor(testLogger(t), testCtx(t), db)
	o, err := p.CreateFromSeed(testDefinition(t, "CRIMSON_BALROG"), registry.Seed{
		Stage: "ATTACKING", WorldId: 1, ChannelId: 4, ConcurrencyKey: "k",
		// Shape matches what CRIMSON_BALROG actually writes
		// (events/crimsonbalrog/config.go OccurrenceContext): `visual` is an
		// object, and the music key is `backgroundMusic` (no state/subState
		// since B6 — those bytes are resolved per-tenant on the channel
		// side).
		Context: json.RawMessage(`{"visual":{"name":"CONTI_MOVE"},"backgroundMusic":"Bgm04/ArabPirate"}`),
		Maps: []registry.MapScope{
			{MapId: 200090010, Visual: true},
			{MapId: 200090011, Visual: false},
		},
	}, "w")
	require.NoError(t, err)

	deck := doGET[VisualRestModel](t, db, "/events/worlds/1/channels/4/maps/200090010/visuals")
	if len(deck) != 1 || deck[0].OccurrenceId != o.Id().String() {
		t.Fatalf("deck returned %d visuals, want 1", len(deck))
	}
	if deck[0].Visual != "CONTI_MOVE" || deck[0].Bgm != "Bgm04/ArabPirate" {
		t.Fatalf("projection = %+v", deck[0])
	}

	cabin := doGET[VisualRestModel](t, db, "/events/worlds/1/channels/4/maps/200090011/visuals")
	if len(cabin) != 0 {
		t.Fatalf("cabin returned %d visuals, want 0", len(cabin))
	}
}

// A COMPLETED occurrence's visual is gone even though its map rows remain.
func TestVisualsProjectionExcludesCompletedOccurrences(t *testing.T) {
	db := newTestDB(t)
	p := NewProcessor(testLogger(t), testCtx(t), db)
	o, err := p.CreateFromSeed(testDefinition(t, "CRIMSON_BALROG"), registry.Seed{
		Stage: "ATTACKING", WorldId: 1, ChannelId: 4, ConcurrencyKey: "k",
		Context: json.RawMessage(`{"visual":{"name":"CONTI_MOVE"},"backgroundMusic":"Bgm04/ArabPirate"}`),
		Maps: []registry.MapScope{
			{MapId: 200090010, Visual: true},
		},
	}, "w")
	require.NoError(t, err)

	before := doGET[VisualRestModel](t, db, "/events/worlds/1/channels/4/maps/200090010/visuals")
	require.Len(t, before, 1)

	won, err := p.Complete(o.Id(), "MONSTERS_ELIMINATED", "MONSTER_KILLED", "u1")
	require.NoError(t, err)
	require.True(t, won)

	after := doGET[VisualRestModel](t, db, "/events/worlds/1/channels/4/maps/200090010/visuals")
	if len(after) != 0 {
		t.Fatalf("after completion returned %d visuals, want 0", len(after))
	}
}

// FR-API5: transitions come back as an included relationship.
func TestOccurrenceDetailIncludesTransitions(t *testing.T) {
	db := newTestDB(t)
	p := NewProcessor(testLogger(t), testCtx(t), db)
	o, err := p.CreateFromSeed(testDefinition(t, "CRIMSON_BALROG"), registry.Seed{
		Stage: "ATTACKING", WorldId: 1, ChannelId: 4, ConcurrencyKey: "k",
	}, "w")
	require.NoError(t, err)

	_, err = p.ApplyProgress(o, registry.Progress{Stage: "FLEEING"}, "SCHEDULED_WORK", "work-2")
	require.NoError(t, err)

	srv := httptest.NewServer(setupOccurrenceRouter(db))
	defer srv.Close()

	url := fmt.Sprintf("%s/events/occurrences/%s", srv.URL, o.Id())
	req := requestWithTenant(http.MethodGet, url, testTenantId)
	resp, err := (&http.Client{}).Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var doc jsonapi.Document
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))
	require.NotNil(t, doc.Data)
	require.NotNil(t, doc.Data.DataObject)
	require.Equal(t, o.Id().String(), doc.Data.DataObject.ID)

	require.Len(t, doc.Included, 2, "expected OCCURRENCE_CREATED + SCHEDULED_WORK transitions")
	for _, inc := range doc.Included {
		require.Equal(t, "event-occurrence-transitions", inc.Type)
	}
}

// FR-API9: another tenant's occurrences are never returned.
func TestOccurrenceListIsTenantScoped(t *testing.T) {
	db := newTestDB(t)
	p := NewProcessor(testLogger(t), testCtx(t), db)
	_, err := p.CreateFromSeed(testDefinition(t, "CRIMSON_BALROG"), registry.Seed{
		Stage: "ATTACKING", WorldId: 1, ChannelId: 4, ConcurrencyKey: "k",
	}, "w")
	require.NoError(t, err)

	mine := doGETAsTenant[RestModel](t, db, "/events/occurrences?page[size]=10", testTenantId)
	if len(mine) != 1 {
		t.Fatalf("own tenant returned %d occurrences, want 1", len(mine))
	}

	other := doGETAsTenant[RestModel](t, db, "/events/occurrences?page[size]=10", uuid.New())
	if len(other) != 0 {
		t.Fatalf("other tenant returned %d occurrences, want 0", len(other))
	}
}
