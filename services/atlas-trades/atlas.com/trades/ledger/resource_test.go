package ledger

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/asset"
	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/miniroom"
)

type testServerInformation struct{}

func (t *testServerInformation) GetBaseURL() string { return "http://localhost:8080" }

func (t *testServerInformation) GetPrefix() string { return "/api/" }

var _ jsonapi.ServerInformation = &testServerInformation{}

// testRouter mounts the real ledger resource, so every test drives the routes,
// the tenant decorator and the JSON:API marshalling exactly as the service does.
func testRouter(t *testing.T, db *gorm.DB) *mux.Router {
	t.Helper()
	r := mux.NewRouter().PathPrefix("/api").Subrouter()
	l := logrus.New()
	l.SetLevel(logrus.PanicLevel)
	InitResource(&testServerInformation{})(db)(r, l)
	return r
}

// restRequest builds a GET carrying the four headers ParseTenant requires. The
// tenant id is the one the ledger's rows are written under, so the handler's
// tenant-scoped query resolves to them.
func restRequest(t *testing.T, url string, tenantId uuid.UUID) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("TENANT_ID", tenantId.String())
	req.Header.Set("REGION", "GMS")
	req.Header.Set("MAJOR_VERSION", "83")
	req.Header.Set("MINOR_VERSION", "1")
	return req
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

// TestGetLedgerEntriesMatchesEitherSide pins FR-7.2 through the REST surface:
// filter[characterId] finds the trade whether the character gave or received.
func TestGetLedgerEntriesMatchesEitherSide(t *testing.T) {
	db := testDb(t)
	tenantId := testTenantId(t)
	stored := recordTrade(t, db, tenantId, time.Now(), 100, 200)
	router := testRouter(t, db)

	for _, id := range []string{"100", "200"} {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, restRequest(t, "/api/trades/ledger?filter[characterId]="+id, tenantId))
		if rec.Code != http.StatusOK {
			t.Fatalf("characterId %s: status %d (body %s)", id, rec.Code, rec.Body.String())
		}
		got := idsOf(t, rec.Body.Bytes())
		if len(got) != 1 || got[0] != stored.Id().String() {
			t.Errorf("characterId %s: got %v, want [%s]", id, got, stored.Id())
		}
	}

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, restRequest(t, "/api/trades/ledger?filter[characterId]=300", tenantId))
	if got := idsOf(t, rec.Body.Bytes()); len(got) != 0 {
		t.Errorf("uninvolved character: got %v, want []", got)
	}
}

// TestGetLedgerEntriesFiltersOnTimeRange pins filter[from]/filter[to] RFC3339
// parsing and the inclusive window they describe.
func TestGetLedgerEntriesFiltersOnTimeRange(t *testing.T) {
	db := testDb(t)
	tenantId := testTenantId(t)
	base := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	older := recordTrade(t, db, tenantId, base.Add(-2*time.Hour), 100, 200)
	newer := recordTrade(t, db, tenantId, base, 100, 200)
	router := testRouter(t, db)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, restRequest(t,
		"/api/trades/ledger?filter[characterId]=100"+
			"&filter[from]="+base.Add(-time.Hour).Format(time.RFC3339)+
			"&filter[to]="+base.Add(time.Hour).Format(time.RFC3339), tenantId))
	if rec.Code != http.StatusOK {
		t.Fatalf("windowed: status %d (body %s)", rec.Code, rec.Body.String())
	}
	got := idsOf(t, rec.Body.Bytes())
	if len(got) != 1 || got[0] != newer.Id().String() {
		t.Fatalf("windowed: got %v, want only [%s]", got, newer.Id())
	}

	// Unwindowed, both entries are in range and newest comes first.
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, restRequest(t, "/api/trades/ledger?filter[characterId]=100", tenantId))
	got = idsOf(t, rec.Body.Bytes())
	if len(got) != 2 || got[0] != newer.Id().String() || got[1] != older.Id().String() {
		t.Errorf("unwindowed: got %v, want [%s %s]", got, newer.Id(), older.Id())
	}
}

// TestGetLedgerEntriesRejectsMalformedFilters pins PRD §5's error table: a
// filter that does not parse is a 400 rather than a silently ignored filter
// that widens the result set.
func TestGetLedgerEntriesRejectsMalformedFilters(t *testing.T) {
	db := testDb(t)
	tenantId := testTenantId(t)
	router := testRouter(t, db)

	for _, query := range []string{
		"",
		"filter[characterId]=abc",
		"filter[characterId]=-1",
		"filter[characterId]=4294967296",
		"filter[characterId]=100&filter[from]=yesterday",
		"filter[characterId]=100&filter[to]=2026-08-09",
		"filter[characterId]=100&page[size]=500",
	} {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, restRequest(t, "/api/trades/ledger?"+query, tenantId))
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%q: got %d, want 400 (body %s)", query, rec.Code, rec.Body.String())
		}
	}
}

// TestGetLedgerEntriesPageSizeCapIs100 pins the cap's exact value: 100 is
// accepted and 101 is not, so a later edit to paginate.MaxPageSize cannot
// silently widen it.
func TestGetLedgerEntriesPageSizeCapIs100(t *testing.T) {
	db := testDb(t)
	tenantId := testTenantId(t)
	router := testRouter(t, db)

	for size, want := range map[string]int{"100": http.StatusOK, "101": http.StatusBadRequest} {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, restRequest(t, "/api/trades/ledger?filter[characterId]=100&page[size]="+size, tenantId))
		if rec.Code != want {
			t.Errorf("page[size]=%s: got %d, want %d", size, rec.Code, want)
		}
	}
}

// TestGetLedgerEntriesPaginates pins the page slice and its envelope: page 2 of
// size 1 holds the older of the two trades, and meta.total counts both.
func TestGetLedgerEntriesPaginates(t *testing.T) {
	db := testDb(t)
	tenantId := testTenantId(t)
	base := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	older := recordTrade(t, db, tenantId, base.Add(-2*time.Hour), 100, 200)
	newer := recordTrade(t, db, tenantId, base, 100, 200)
	router := testRouter(t, db)

	for page, want := range map[string]uuid.UUID{"1": newer.Id(), "2": older.Id()} {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, restRequest(t,
			"/api/trades/ledger?filter[characterId]=100&page[number]="+page+"&page[size]=1", tenantId))
		if rec.Code != http.StatusOK {
			t.Fatalf("page %s: status %d (body %s)", page, rec.Code, rec.Body.String())
		}
		got := idsOf(t, rec.Body.Bytes())
		if len(got) != 1 || got[0] != want.String() {
			t.Fatalf("page %s: got %v, want [%s]", page, got, want)
		}

		var doc struct {
			Meta struct {
				Total int `json:"total"`
			} `json:"meta"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &doc); err != nil {
			t.Fatalf("page %s: decode meta: %v", page, err)
		}
		if doc.Meta.Total != 2 {
			t.Errorf("page %s: meta.total got %d, want 2", page, doc.Meta.Total)
		}
	}
}

// TestGetLedgerEntriesIsTenantScoped is the cross-tenant guard on the list read:
// two tenants each record a trade for the SAME character id, and each tenant's
// request must return only its own entry.
func TestGetLedgerEntriesIsTenantScoped(t *testing.T) {
	db := testDb(t)
	tenantA := testTenantId(t)
	tenantB := uuid.New()
	settledAt := time.Now()
	mine := recordTrade(t, db, tenantA, settledAt, 100, 200)
	theirs := recordTrade(t, db, tenantB, settledAt, 100, 200)
	router := testRouter(t, db)

	for name, tc := range map[string]struct {
		tenantId uuid.UUID
		want     uuid.UUID
	}{
		"tenant A": {tenantA, mine.Id()},
		"tenant B": {tenantB, theirs.Id()},
	} {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, restRequest(t, "/api/trades/ledger?filter[characterId]=100", tc.tenantId))
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: status %d (body %s)", name, rec.Code, rec.Body.String())
		}
		got := idsOf(t, rec.Body.Bytes())
		if len(got) != 1 || got[0] != tc.want.String() {
			t.Errorf("%s: got %v, want [%s]", name, got, tc.want)
		}
	}
}

// TestGetLedgerEntryByIdServesTheEntry pins the single-entry projection: both
// sides, their meso columns and their items reach the wire.
func TestGetLedgerEntryByIdServesTheEntry(t *testing.T) {
	db := testDb(t)
	tenantId := testTenantId(t)
	assetId := asset.Id(9001)
	referenceId := uint32(55)
	entry := NewBuilder(uuid.New(), testField(t), miniroom.CashTrade).
		SetSettledAt(time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)).
		AddSide(100, "Alice", 10_000, 400, 0, []Item{
			NewItem(2000000, 5, nil, nil),
			NewItem(1302000, 1, &assetId, &referenceId),
		}).
		AddSide(200, "Bob", 0, 0, 9_600, nil).
		Build()
	stored, err := create(db, tenantId)(entry)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	router := testRouter(t, db)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, restRequest(t, "/api/trades/ledger/"+stored.Id().String(), tenantId))
	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200 (body %s)", rec.Code, rec.Body.String())
	}

	var doc struct {
		Data struct {
			Id         string `json:"id"`
			Type       string `json:"type"`
			Attributes struct {
				TransactionId string `json:"transactionId"`
				WorldId       byte   `json:"worldId"`
				ChannelId     byte   `json:"channelId"`
				MapId         uint32 `json:"mapId"`
				RoomType      byte   `json:"roomType"`
				Sides         []struct {
					CharacterId   uint32 `json:"characterId"`
					CharacterName string `json:"characterName"`
					MesoStaged    uint32 `json:"mesoStaged"`
					MesoTax       uint32 `json:"mesoTax"`
					MesoDelivered uint32 `json:"mesoDelivered"`
					Items         []struct {
						ItemId      uint32  `json:"itemId"`
						Quantity    uint32  `json:"quantity"`
						AssetId     *uint32 `json:"assetId"`
						ReferenceId *uint32 `json:"referenceId"`
					} `json:"items"`
				} `json:"sides"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &doc); err != nil {
		t.Fatalf("decode document: %v (body %s)", err, rec.Body.String())
	}
	if doc.Data.Type != "ledgerEntries" {
		t.Errorf("resource type: got %q, want %q", doc.Data.Type, "ledgerEntries")
	}
	if doc.Data.Id != stored.Id().String() {
		t.Errorf("id: got %q, want %q", doc.Data.Id, stored.Id())
	}
	if doc.Data.Attributes.TransactionId != entry.TransactionId().String() {
		t.Errorf("transactionId: got %q, want %q", doc.Data.Attributes.TransactionId, entry.TransactionId())
	}
	if doc.Data.Attributes.RoomType != miniroom.CashTrade {
		t.Errorf("roomType: got %d, want %d", doc.Data.Attributes.RoomType, miniroom.CashTrade)
	}
	if doc.Data.Attributes.WorldId != 1 || doc.Data.Attributes.ChannelId != 2 || doc.Data.Attributes.MapId != 100000000 {
		t.Errorf("field: got %d/%d/%d, want 1/2/100000000",
			doc.Data.Attributes.WorldId, doc.Data.Attributes.ChannelId, doc.Data.Attributes.MapId)
	}
	if len(doc.Data.Attributes.Sides) != 2 {
		t.Fatalf("sides: got %d, want 2", len(doc.Data.Attributes.Sides))
	}
	// Sides are ordered by character id (Model.Sides' determinism guarantee).
	alice := doc.Data.Attributes.Sides[0]
	if alice.CharacterId != 100 || alice.CharacterName != "Alice" {
		t.Errorf("first side: got %d/%q, want 100/Alice", alice.CharacterId, alice.CharacterName)
	}
	if alice.MesoStaged != 10_000 || alice.MesoTax != 400 || alice.MesoDelivered != 0 {
		t.Errorf("Alice's meso: got %d/%d/%d, want 10000/400/0", alice.MesoStaged, alice.MesoTax, alice.MesoDelivered)
	}
	if len(alice.Items) != 2 {
		t.Fatalf("Alice's items: got %d, want 2", len(alice.Items))
	}
	// Items are ordered by item id.
	if alice.Items[0].ItemId != 1302000 || alice.Items[0].AssetId == nil || *alice.Items[0].AssetId != 9001 {
		t.Errorf("equip item: got %+v, want itemId 1302000 with assetId 9001", alice.Items[0])
	}
	if alice.Items[0].ReferenceId == nil || *alice.Items[0].ReferenceId != referenceId {
		t.Errorf("equip referenceId: got %v, want %d", alice.Items[0].ReferenceId, referenceId)
	}
	if alice.Items[1].ItemId != 2000000 || alice.Items[1].Quantity != 5 {
		t.Errorf("stackable item: got %+v, want itemId 2000000 quantity 5", alice.Items[1])
	}
	if alice.Items[1].AssetId != nil || alice.Items[1].ReferenceId != nil {
		t.Errorf("stackable item must carry no identity: got %+v", alice.Items[1])
	}
}

// TestGetLedgerEntryByIdReturns404WhenUnknown pins PRD §5's error table: an
// unknown entry id is a 404, not a 500.
func TestGetLedgerEntryByIdReturns404WhenUnknown(t *testing.T) {
	db := testDb(t)
	router := testRouter(t, db)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, restRequest(t, "/api/trades/ledger/"+uuid.New().String(), testTenantId(t)))

	if rec.Code != http.StatusNotFound {
		t.Errorf("status: got %d, want 404 (body %s)", rec.Code, rec.Body.String())
	}
}

// TestGetLedgerEntryByIdRejectsMalformedId pins PRD §5's error table: an entry
// id that is not a uuid is a client error, not a 404.
func TestGetLedgerEntryByIdRejectsMalformedId(t *testing.T) {
	db := testDb(t)
	router := testRouter(t, db)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, restRequest(t, "/api/trades/ledger/not-a-uuid", testTenantId(t)))

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", rec.Code)
	}
}

// TestGetLedgerEntryByIdIsTenantScoped is the cross-tenant guard on the
// single-entry read: one tenant must not be able to read another's entry by
// guessing its id. Dropping the tenant filter from the query serves it.
func TestGetLedgerEntryByIdIsTenantScoped(t *testing.T) {
	db := testDb(t)
	tenantA := testTenantId(t)
	tenantB := uuid.New()
	theirs := recordTrade(t, db, tenantB, time.Now(), 100, 200)
	router := testRouter(t, db)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, restRequest(t, "/api/trades/ledger/"+theirs.Id().String(), tenantA))
	if rec.Code != http.StatusNotFound {
		t.Errorf("tenant A read of tenant B's entry: got %d, want 404 (body %s)", rec.Code, rec.Body.String())
	}

	// The owning tenant must still read it, so the assertion above is about the
	// tenant filter and not about the entry being unreadable.
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, restRequest(t, "/api/trades/ledger/"+theirs.Id().String(), tenantB))
	if rec.Code != http.StatusOK {
		t.Errorf("owning tenant read: got %d, want 200 (body %s)", rec.Code, rec.Body.String())
	}
}

// TestGetLedgerEntriesRequiresTenantHeaders pins PRD §5's error table: a request
// with no tenant header is a 400, never an unscoped read.
func TestGetLedgerEntriesRequiresTenantHeaders(t *testing.T) {
	db := testDb(t)
	router := testRouter(t, db)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/trades/ledger?filter[characterId]=100", nil))

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", rec.Code)
	}
}

// TestLedgerRejectsWrites pins FR-7.4: ledger rows are immutable, so no write
// verb is routed. gorilla/mux answers an unrouted method on a subrouter path
// with 404 rather than 405, so the assertion is that the request is refused,
// not which of the two refusals it gets.
func TestLedgerRejectsWrites(t *testing.T) {
	db := testDb(t)
	router := testRouter(t, db)

	for _, method := range []string{http.MethodPost, http.MethodPatch, http.MethodDelete} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(method, "/api/trades/ledger", nil)
		req.Header.Set("TENANT_ID", testTenantId(t).String())
		req.Header.Set("REGION", "GMS")
		req.Header.Set("MAJOR_VERSION", "83")
		req.Header.Set("MINOR_VERSION", "1")
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusMethodNotAllowed && rec.Code != http.StatusNotFound {
			t.Errorf("%s /trades/ledger: got %d, want 404 or 405", method, rec.Code)
		}
	}
}

// TestTransformProjectsOptionalAssetIdentity pins that an item with no identity
// of its own renders null rather than 0, which a consumer would read as a real
// asset id.
func TestTransformProjectsOptionalAssetIdentity(t *testing.T) {
	assetId := asset.Id(9001)
	referenceId := uint32(55)
	m := NewBuilder(uuid.New(), testField(t), miniroom.Trade).
		AddSide(character.Id(100), "Alice", 0, 0, 0, []Item{
			NewItem(2000000, 5, nil, nil),
			NewItem(1302000, 1, &assetId, &referenceId),
		}).
		AddSide(character.Id(200), "Bob", 0, 0, 0, nil).
		Build()

	rm, err := Transform(m)
	if err != nil {
		t.Fatalf("transform: %v", err)
	}
	if len(rm.Sides) != 2 || len(rm.Sides[0].Items) != 2 {
		t.Fatalf("shape: got %d sides and %d items on the first", len(rm.Sides), len(rm.Sides[0].Items))
	}
	plain := rm.Sides[0].Items[0]
	if plain.AssetId != nil || plain.ReferenceId != nil {
		t.Errorf("stackable item: got assetId %v / referenceId %v, want both nil", plain.AssetId, plain.ReferenceId)
	}
	equip := rm.Sides[0].Items[1]
	if equip.AssetId == nil || *equip.AssetId != assetId {
		t.Errorf("equip assetId: got %v, want %d", equip.AssetId, assetId)
	}
	if equip.ReferenceId == nil || *equip.ReferenceId != referenceId {
		t.Errorf("equip referenceId: got %v, want %d", equip.ReferenceId, referenceId)
	}
}
