package trade

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/miniroom"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type testServerInformation struct{}

func (t *testServerInformation) GetBaseURL() string { return "http://localhost:8080" }

func (t *testServerInformation) GetPrefix() string { return "/api/" }

var _ jsonapi.ServerInformation = &testServerInformation{}

// testRouter mounts the real room resource, so every test drives the routes,
// the tenant decorator and the JSON:API marshalling exactly as the service does.
func testRouter(t *testing.T) *mux.Router {
	t.Helper()
	r := mux.NewRouter().PathPrefix("/api").Subrouter()
	l := logrus.New()
	l.SetLevel(logrus.PanicLevel)
	InitResource(&testServerInformation{})(r, l)
	return r
}

// restTenantId derives a tenant id from the test's name, so tests sharing the
// process-wide registry singleton cannot see each other's rooms.
func restTenantId(t *testing.T) uuid.UUID {
	t.Helper()
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(t.Name()))
}

// restOtherTenantId is a second, distinct tenant id for the same test.
func restOtherTenantId(t *testing.T) uuid.UUID {
	t.Helper()
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(t.Name()+"-other"))
}

// restTenant is the tenant.Model the tenant decorator builds from the headers
// restRequest sets, so a room registered under it is the one a handler reads.
func restTenant(t *testing.T, id uuid.UUID) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(id, "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create() error = %v", err)
	}
	return tm
}

// restRequest builds a GET carrying the four headers ParseTenant requires.
func restRequest(t *testing.T, url string, tenantId uuid.UUID) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("TENANT_ID", tenantId.String())
	req.Header.Set("REGION", "GMS")
	req.Header.Set("MAJOR_VERSION", "83")
	req.Header.Set("MINOR_VERSION", "1")
	return req
}

// registerRoom puts a room in the process-wide registry for the duration of the
// test.
func registerRoom(t *testing.T, tm tenant.Model, r Room) Room {
	t.Helper()
	if err := GetRegistry().Create(tm, r); err != nil {
		t.Fatalf("register room: %v", err)
	}
	t.Cleanup(func() { GetRegistry().Remove(tm, r.Id()) })
	return r
}

// restField is the world/channel/map the test rooms live in.
func restField(t *testing.T) field.Model {
	t.Helper()
	return field.NewBuilder(world.Id(1), channel.Id(2), _map.Id(100000000)).Build()
}

// idsOf returns the resource ids in a JSON:API document's data array.
func idsOf(t *testing.T, body []byte) []string {
	t.Helper()
	var doc struct {
		Data []struct {
			Id string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		t.Fatalf("decode document: %v (body %s)", err, string(body))
	}
	out := make([]string, 0, len(doc.Data))
	for _, d := range doc.Data {
		out = append(out, d.Id)
	}
	return out
}

// TestGetRoomByIdReturns404WhenGone pins PRD §5: a settled or cancelled room is
// gone, and the endpoint must 404 rather than serve a stale snapshot.
func TestGetRoomByIdReturns404WhenGone(t *testing.T) {
	router := testRouter(t)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, restRequest(t, "/api/trades/rooms/"+uuid.New().String(), restTenantId(t)))

	if rec.Code != http.StatusNotFound {
		t.Errorf("status: got %d, want 404", rec.Code)
	}
}

// TestGetRoomByIdRejectsMalformedId pins PRD §5's error table: a room id that is
// not a uuid is a client error, not a 404.
func TestGetRoomByIdRejectsMalformedId(t *testing.T) {
	router := testRouter(t)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, restRequest(t, "/api/trades/rooms/not-a-uuid", restTenantId(t)))

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", rec.Code)
	}
}

// TestGetRoomByIdServesTheRoom pins the happy path: the single-room read
// projects the live room, its participants and their staged items.
func TestGetRoomByIdServesTheRoom(t *testing.T) {
	router := testRouter(t)
	tm := restTenant(t, restTenantId(t))
	room := registerRoom(t, tm, NewBuilder(miniroom.Trade, 100, "Owner", restField(t)).
		SetVisitor(200, "Guest").
		SetState(StateOpen).
		Build())

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, restRequest(t, "/api/trades/rooms/"+room.Id().String(), restTenantId(t)))

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200 (body %s)", rec.Code, rec.Body.String())
	}
	var doc struct {
		Data struct {
			Id         string `json:"id"`
			Type       string `json:"type"`
			Attributes struct {
				RoomType     byte   `json:"roomType"`
				WorldId      byte   `json:"worldId"`
				ChannelId    byte   `json:"channelId"`
				MapId        uint32 `json:"mapId"`
				State        string `json:"state"`
				Participants []struct {
					CharacterId uint32 `json:"characterId"`
					Position    byte   `json:"position"`
				} `json:"participants"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &doc); err != nil {
		t.Fatalf("decode document: %v (body %s)", err, rec.Body.String())
	}
	if doc.Data.Type != "rooms" {
		t.Errorf("resource type: got %q, want %q", doc.Data.Type, "rooms")
	}
	if doc.Data.Id != room.Id().String() {
		t.Errorf("id: got %q, want %q", doc.Data.Id, room.Id().String())
	}
	if doc.Data.Attributes.RoomType != miniroom.Trade {
		t.Errorf("roomType: got %d, want %d", doc.Data.Attributes.RoomType, miniroom.Trade)
	}
	if doc.Data.Attributes.State != string(StateOpen) {
		t.Errorf("state: got %q, want %q", doc.Data.Attributes.State, StateOpen)
	}
	if doc.Data.Attributes.WorldId != 1 || doc.Data.Attributes.ChannelId != 2 || doc.Data.Attributes.MapId != 100000000 {
		t.Errorf("field: got %d/%d/%d, want 1/2/100000000",
			doc.Data.Attributes.WorldId, doc.Data.Attributes.ChannelId, doc.Data.Attributes.MapId)
	}
	if len(doc.Data.Attributes.Participants) != 2 {
		t.Fatalf("participants: got %d, want 2", len(doc.Data.Attributes.Participants))
	}
}

// TestGetRoomByIdIsTenantScoped pins the NFR on the single-room read: tenant B
// must not be able to read tenant A's room by its id. Dropping the tenant from
// the registry lookup serves it.
func TestGetRoomByIdIsTenantScoped(t *testing.T) {
	router := testRouter(t)
	tm := restTenant(t, restTenantId(t))
	room := registerRoom(t, tm, NewBuilder(miniroom.Trade, 100, "Owner", restField(t)).Build())

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, restRequest(t, "/api/trades/rooms/"+room.Id().String(), restOtherTenantId(t)))

	if rec.Code != http.StatusNotFound {
		t.Errorf("tenant B read of tenant A's room: got %d, want 404 (body %s)", rec.Code, rec.Body.String())
	}
}

// TestGetRoomsIsTenantScoped pins the NFR: a room in tenant A must be invisible
// from tenant B.
func TestGetRoomsIsTenantScoped(t *testing.T) {
	router := testRouter(t)
	tm := restTenant(t, restTenantId(t))
	room := registerRoom(t, tm, NewBuilder(miniroom.Trade, 100, "Owner", restField(t)).Build())

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, restRequest(t, "/api/trades/rooms", restOtherTenantId(t)))

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200 (body %s)", rec.Code, rec.Body.String())
	}
	if body := rec.Body.String(); strings.Contains(body, room.Id().String()) {
		t.Errorf("tenant B sees tenant A's room: %s", body)
	}

	// The same request under tenant A must see it, so the assertion above is
	// about the tenant filter and not about the room never being served.
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, restRequest(t, "/api/trades/rooms", restTenantId(t)))
	if got := idsOf(t, rec.Body.Bytes()); len(got) != 1 || got[0] != room.Id().String() {
		t.Errorf("tenant A list: got %v, want [%s]", got, room.Id())
	}
}

// TestGetRoomsRejectsOversizePage pins PRD §5's error table: a page size above
// the cap is a 400, not a silent clamp.
func TestGetRoomsRejectsOversizePage(t *testing.T) {
	router := testRouter(t)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, restRequest(t, "/api/trades/rooms?page[size]=500", restTenantId(t)))

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", rec.Code)
	}
}

// TestGetRoomsPageSizeCapIs100 pins the cap's exact value: 100 is accepted and
// 101 is not, so a later edit to paginate.MaxPageSize cannot silently widen it.
func TestGetRoomsPageSizeCapIs100(t *testing.T) {
	router := testRouter(t)

	for size, want := range map[string]int{"100": http.StatusOK, "101": http.StatusBadRequest} {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, restRequest(t, "/api/trades/rooms?page[size]="+size, restTenantId(t)))
		if rec.Code != want {
			t.Errorf("page[size]=%s: got %d, want %d", size, rec.Code, want)
		}
	}
}

// TestGetRoomsPaginatesDeterministically pins the sort-then-slice contract: the
// registry's map iteration order is random, so without the sort the same page
// would return different rooms on different requests.
func TestGetRoomsPaginatesDeterministically(t *testing.T) {
	router := testRouter(t)
	tm := restTenant(t, restTenantId(t))

	ids := make([]string, 0, 3)
	for i := character.Id(1); i <= 3; i++ {
		room := registerRoom(t, tm, NewBuilder(miniroom.Trade, 100*i, "Owner", restField(t)).Build())
		ids = append(ids, room.Id().String())
	}
	sortedIds := append([]string(nil), ids...)
	for i := range sortedIds {
		for j := i + 1; j < len(sortedIds); j++ {
			if sortedIds[j] < sortedIds[i] {
				sortedIds[i], sortedIds[j] = sortedIds[j], sortedIds[i]
			}
		}
	}

	for page, want := range map[string][]string{
		"1": {sortedIds[0], sortedIds[1]},
		"2": {sortedIds[2]},
	} {
		for attempt := 0; attempt < 3; attempt++ {
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, restRequest(t, "/api/trades/rooms?page[number]="+page+"&page[size]=2", restTenantId(t)))
			if rec.Code != http.StatusOK {
				t.Fatalf("page %s: status %d (body %s)", page, rec.Code, rec.Body.String())
			}
			got := idsOf(t, rec.Body.Bytes())
			if len(got) != len(want) {
				t.Fatalf("page %s attempt %d: got %v, want %v", page, attempt, got, want)
			}
			for i := range got {
				if got[i] != want[i] {
					t.Fatalf("page %s attempt %d: got %v, want %v", page, attempt, got, want)
				}
			}
		}
	}
}

// TestGetRoomsFilters pins PRD §5's four room filters, each in its matching and
// its non-matching case.
func TestGetRoomsFilters(t *testing.T) {
	router := testRouter(t)
	tm := restTenant(t, restTenantId(t))
	room := registerRoom(t, tm, NewBuilder(miniroom.Trade, 100, "Owner", restField(t)).
		SetVisitor(200, "Guest").
		Build())

	for query, want := range map[string]bool{
		"filter[characterId]=100": true,
		"filter[characterId]=200": true,
		"filter[characterId]=300": false,
		"filter[worldId]=1":       true,
		"filter[worldId]=2":       false,
		"filter[channelId]=2":     true,
		"filter[channelId]=3":     false,
		"filter[mapId]=100000000": true,
		"filter[mapId]=200000000": false,
	} {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, restRequest(t, "/api/trades/rooms?"+query, restTenantId(t)))
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: status %d (body %s)", query, rec.Code, rec.Body.String())
		}
		got := idsOf(t, rec.Body.Bytes())
		matched := len(got) == 1 && got[0] == room.Id().String()
		if matched != want {
			t.Errorf("%s: matched=%t, want %t (body %s)", query, matched, want, rec.Body.String())
		}
	}
}

// TestGetRoomsRejectsMalformedFilters pins PRD §5's error table: a filter that
// is not a number, or is out of its field's range, is a 400 rather than a
// silently ignored filter that returns every room.
func TestGetRoomsRejectsMalformedFilters(t *testing.T) {
	router := testRouter(t)

	for _, query := range []string{
		"filter[characterId]=abc",
		"filter[characterId]=-1",
		"filter[worldId]=256",
		"filter[channelId]=abc",
		"filter[mapId]=4294967296",
	} {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, restRequest(t, "/api/trades/rooms?"+query, restTenantId(t)))
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s: got %d, want 400", query, rec.Code)
		}
	}
}

// TestGetRoomsRequiresTenantHeaders pins PRD §5's error table: a request with no
// tenant header is a 400, never an unscoped read.
func TestGetRoomsRequiresTenantHeaders(t *testing.T) {
	router := testRouter(t)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/trades/rooms", nil))

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", rec.Code)
	}
}

// TestRoomsRejectWrites pins PRD §5: rooms are Kafka-driven, so no write verb
// is routed. gorilla/mux answers an unrouted method on a subrouter path with
// 404 rather than 405, so the assertion is that the request is refused, not
// which of the two refusals it gets.
func TestRoomsRejectWrites(t *testing.T) {
	router := testRouter(t)

	for _, method := range []string{http.MethodPost, http.MethodPatch, http.MethodDelete} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(method, "/api/trades/rooms", nil)
		req.Header.Set("TENANT_ID", restTenantId(t).String())
		req.Header.Set("REGION", "GMS")
		req.Header.Set("MAJOR_VERSION", "83")
		req.Header.Set("MINOR_VERSION", "1")
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusMethodNotAllowed && rec.Code != http.StatusNotFound {
			t.Errorf("%s /trades/rooms: got %d, want 404 or 405", method, rec.Code)
		}
	}
}

// TestTransformProjectsStagedItems pins the staged-item projection: the trade
// dialog slot, the template, the quantity and the asset identity all reach the
// wire.
func TestTransformProjectsStagedItems(t *testing.T) {
	room := NewBuilder(miniroom.Trade, 100, "Owner", restField(t)).Build().
		WithParticipant(0, func(p Participant) Participant {
			return p.WithMesoStaged(1_000).WithItem(NewStagedItem(3, 9001, 1302000, 1, 1, 7))
		})

	rm, err := Transform(room)
	if err != nil {
		t.Fatalf("transform: %v", err)
	}
	if len(rm.Participants) != 1 {
		t.Fatalf("participants: got %d, want 1", len(rm.Participants))
	}
	p := rm.Participants[0]
	if p.MesoStaged != 1_000 {
		t.Errorf("mesoStaged: got %d, want 1000", p.MesoStaged)
	}
	if len(p.Items) != 1 {
		t.Fatalf("items: got %d, want 1", len(p.Items))
	}
	i := p.Items[0]
	if i.TradeSlot != 3 || i.AssetId != 9001 || i.ItemId != 1302000 || i.Quantity != 1 {
		t.Errorf("item: got %+v, want tradeSlot 3 / assetId 9001 / itemId 1302000 / quantity 1", i)
	}
}
