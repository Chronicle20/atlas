package templates

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

func TestReseedReturnsNoContent(t *testing.T) {
	db := setupTestDB(t)
	l := testLogger()
	InitShippedCatalog(l, seedTemplatesDir())

	catalog := ShippedCatalog()
	entry, ok := catalog.Lookup("GMS", 83, 1)
	if !ok {
		t.Fatalf("GMS 83.1 missing from the seed corpus")
	}

	p := NewProcessor(l, context.Background(), db).WithCatalog(catalog)
	id, err := p.Create(entry.Model)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Drift the row so the re-seed has something to undo.
	mutated := entry.Model
	mutated.UsesPin = !mutated.UsesPin
	data, err := canonicalBytes(mutated)
	if err != nil {
		t.Fatalf("canonicalBytes: %v", err)
	}
	if err := db.Model(&Entity{}).Where("id = ?", id).Update("data", []byte(data)).Error; err != nil {
		t.Fatalf("mutate: %v", err)
	}

	router := mux.NewRouter()
	InitResource(testServerInformation{})(db)(router, l)

	req := httptest.NewRequest(http.MethodPost, "/configurations/templates/"+id.String()+"/reseed", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204, body=%s", rr.Code, rr.Body.String())
	}
	if rr.Body.Len() != 0 {
		t.Fatalf("body = %q, want empty", rr.Body.String())
	}

	v, err := p.ViewByIdProvider(id)()
	if err != nil {
		t.Fatalf("ViewByIdProvider: %v", err)
	}
	if v.SeedDrift {
		t.Errorf("SeedDrift = true after a 204 re-seed")
	}
}

func TestReseedUnknownIdReturns404(t *testing.T) {
	db := setupTestDB(t)
	l := testLogger()
	InitShippedCatalog(l, seedTemplatesDir())

	router := mux.NewRouter()
	InitResource(testServerInformation{})(db)(router, l)

	req := httptest.NewRequest(http.MethodPost, "/configurations/templates/"+uuid.New().String()+"/reseed", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404, body=%s", rr.Code, rr.Body.String())
	}
	assertJSONAPIErrorDocument(t, rr, "404")
}

func TestReseedNoShippedFileReturns409(t *testing.T) {
	db := setupTestDB(t)
	l := testLogger()
	InitShippedCatalog(l, seedTemplatesDir())

	// A region/version the real corpus does not ship.
	p := NewProcessor(l, context.Background(), db).WithCatalog(ShippedCatalog())
	id, err := p.Create(createTestRestModel("NOSUCH", 999, 999))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	router := mux.NewRouter()
	InitResource(testServerInformation{})(db)(router, l)

	req := httptest.NewRequest(http.MethodPost, "/configurations/templates/"+id.String()+"/reseed", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409, body=%s", rr.Code, rr.Body.String())
	}
	assertJSONAPIErrorDocument(t, rr, "409")
}

// The GET-by-id read path must carry the three computed attributes, or the UI
// has nothing to render a badge from.
func TestGetTemplateByIdCarriesDriftAttributes(t *testing.T) {
	db := setupTestDB(t)
	l := testLogger()
	InitShippedCatalog(l, seedTemplatesDir())

	entry, ok := ShippedCatalog().Lookup("GMS", 83, 1)
	if !ok {
		t.Fatalf("GMS 83.1 missing from the seed corpus")
	}
	p := NewProcessor(l, context.Background(), db).WithCatalog(ShippedCatalog())
	id, err := p.Create(entry.Model)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	router := mux.NewRouter()
	InitResource(testServerInformation{})(db)(router, l)

	req := httptest.NewRequest(http.MethodGet, "/configurations/templates/"+id.String(), nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rr.Code, rr.Body.String())
	}

	var doc struct {
		Data struct {
			Type       string         `json:"type"`
			Id         string         `json:"id"`
			Attributes map[string]any `json:"attributes"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &doc); err != nil {
		t.Fatalf("unmarshal response: %v (body=%s)", err, rr.Body.String())
	}
	if doc.Data.Type != "templates" {
		t.Errorf("type = %q, want templates", doc.Data.Type)
	}
	if doc.Data.Id != id.String() {
		t.Errorf("id = %q, want %q", doc.Data.Id, id.String())
	}
	if got := doc.Data.Attributes["shippedRevision"]; got != entry.Revision {
		t.Errorf("shippedRevision = %v, want %q", got, entry.Revision)
	}
	if got := doc.Data.Attributes["storedRevision"]; got != entry.Revision {
		t.Errorf("storedRevision = %v, want %q", got, entry.Revision)
	}
	if got := doc.Data.Attributes["seedDrift"]; got != false {
		t.Errorf("seedDrift = %v, want false", got)
	}
	// The read shape must still carry the ordinary template attributes.
	if _, present := doc.Data.Attributes["socket"]; !present {
		t.Errorf("socket attribute missing - the embedded RestModel did not flatten")
	}
}

func assertJSONAPIErrorDocument(t *testing.T, rr *httptest.ResponseRecorder, wantStatus string) {
	t.Helper()
	if ct := rr.Header().Get("Content-Type"); ct != "application/vnd.api+json" {
		t.Errorf("Content-Type = %q, want application/vnd.api+json", ct)
	}
	var doc struct {
		Errors []struct {
			Status string `json:"status"`
			Title  string `json:"title"`
			Detail string `json:"detail"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &doc); err != nil {
		t.Fatalf("unmarshal error document: %v (body=%s)", err, rr.Body.String())
	}
	if len(doc.Errors) != 1 {
		t.Fatalf("len(errors) = %d, want 1 (body=%s)", len(doc.Errors), rr.Body.String())
	}
	if doc.Errors[0].Status != wantStatus {
		t.Errorf("errors[0].status = %q, want %q", doc.Errors[0].Status, wantStatus)
	}
	if doc.Errors[0].Title == "" || doc.Errors[0].Detail == "" {
		t.Errorf("errors[0] has an empty title or detail: %+v", doc.Errors[0])
	}
}
