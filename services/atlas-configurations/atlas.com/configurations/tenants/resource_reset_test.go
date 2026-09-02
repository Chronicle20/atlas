package tenants

import (
	"atlas-configurations/templates"
	tmplhandler "atlas-configurations/templates/socket/handler"
	tmplwriter "atlas-configurations/templates/socket/writer"
	"atlas-configurations/tenants/npcs"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// assertJSONAPIErrorDocument copied verbatim from
// templates/resource_reseed_test.go:167-191, unexported and package-local
// there.
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

func doReset(t *testing.T, router *mux.Router, id string, body string) *httptest.ResponseRecorder {
	t.Helper()
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req := httptest.NewRequest(http.MethodPost, "/configurations/tenants/"+id+"/reset", r)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

// resetResponseDoc decodes just enough of the JSON:API document to assert
// on the computed drift attributes and the flattened embedded RestModel.
type resetResponseDoc struct {
	Data struct {
		Attributes map[string]json.RawMessage `json:"attributes"`
	} `json:"data"`
}

func decodeResetResponse(t *testing.T, rr *httptest.ResponseRecorder) resetResponseDoc {
	t.Helper()
	var doc resetResponseDoc
	if err := json.Unmarshal(rr.Body.Bytes(), &doc); err != nil {
		t.Fatalf("unmarshal response document: %v (body=%s)", err, rr.Body.String())
	}
	return doc
}

func assertComputedAttributesPresent(t *testing.T, attrs map[string]json.RawMessage) {
	t.Helper()
	for _, key := range []string{"baselineTemplateId", "baselineRevision", "storedRevision", "templateDrift", "sectionDrift", "socket"} {
		if _, ok := attrs[key]; !ok {
			t.Errorf("attributes missing key %q", key)
		}
	}
}

func TestResetEndpoint(t *testing.T) {
	t.Run("AbsentBodyIsWholeDocument", func(t *testing.T) {
		db := setupViewTestDB(t)
		l := testLogger()
		router := mux.NewRouter()
		InitResource(testServerInformation{})(db)(router, l)
		p := NewProcessor(l, context.Background(), db)

		seedTemplate(t, db, "GMS", 83, 1, nil)
		id := seedTenant(t, db, p, "GMS", 83, 1, func(rm *RestModel) {
			rm.UsesPin = false
		})

		rr := doReset(t, router, id.String(), "")
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200, body=%s", rr.Code, rr.Body.String())
		}
		doc := decodeResetResponse(t, rr)
		assertComputedAttributesPresent(t, doc.Data.Attributes)

		var templateDrift bool
		if err := json.Unmarshal(doc.Data.Attributes["templateDrift"], &templateDrift); err != nil {
			t.Fatalf("unmarshal templateDrift: %v", err)
		}
		if templateDrift {
			t.Error("expected templateDrift false")
		}
		var sectionDrift map[string]bool
		if err := json.Unmarshal(doc.Data.Attributes["sectionDrift"], &sectionDrift); err != nil {
			t.Fatalf("unmarshal sectionDrift: %v", err)
		}
		assertAllSectionsFalse(t, sectionDrift)
	})

	t.Run("EmptyObjectIsWholeDocument", func(t *testing.T) {
		db := setupViewTestDB(t)
		l := testLogger()
		router := mux.NewRouter()
		InitResource(testServerInformation{})(db)(router, l)
		p := NewProcessor(l, context.Background(), db)

		seedTemplate(t, db, "GMS", 83, 1, nil)
		id := seedTenant(t, db, p, "GMS", 83, 1, func(rm *RestModel) {
			rm.UsesPin = false
		})

		rr := doReset(t, router, id.String(), "{}")
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200, body=%s", rr.Code, rr.Body.String())
		}
		doc := decodeResetResponse(t, rr)
		assertComputedAttributesPresent(t, doc.Data.Attributes)

		var templateDrift bool
		if err := json.Unmarshal(doc.Data.Attributes["templateDrift"], &templateDrift); err != nil {
			t.Fatalf("unmarshal templateDrift: %v", err)
		}
		if templateDrift {
			t.Error("expected templateDrift false")
		}
		var sectionDrift map[string]bool
		if err := json.Unmarshal(doc.Data.Attributes["sectionDrift"], &sectionDrift); err != nil {
			t.Fatalf("unmarshal sectionDrift: %v", err)
		}
		assertAllSectionsFalse(t, sectionDrift)
	})

	t.Run("AbsentSectionsKeyIsWholeDocument", func(t *testing.T) {
		db := setupViewTestDB(t)
		l := testLogger()
		router := mux.NewRouter()
		InitResource(testServerInformation{})(db)(router, l)
		p := NewProcessor(l, context.Background(), db)

		seedTemplate(t, db, "GMS", 83, 1, nil)
		id := seedTenant(t, db, p, "GMS", 83, 1, func(rm *RestModel) {
			rm.UsesPin = false
		})

		rr := doReset(t, router, id.String(), `{"data":{"type":"tenants","attributes":{}}}`)
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200, body=%s", rr.Code, rr.Body.String())
		}
		doc := decodeResetResponse(t, rr)
		assertComputedAttributesPresent(t, doc.Data.Attributes)

		var templateDrift bool
		if err := json.Unmarshal(doc.Data.Attributes["templateDrift"], &templateDrift); err != nil {
			t.Fatalf("unmarshal templateDrift: %v", err)
		}
		if templateDrift {
			t.Error("expected templateDrift false")
		}
		var sectionDrift map[string]bool
		if err := json.Unmarshal(doc.Data.Attributes["sectionDrift"], &sectionDrift); err != nil {
			t.Fatalf("unmarshal sectionDrift: %v", err)
		}
		assertAllSectionsFalse(t, sectionDrift)
	})

	t.Run("EmptySectionsArrayIsWholeDocument", func(t *testing.T) {
		db := setupViewTestDB(t)
		l := testLogger()
		router := mux.NewRouter()
		InitResource(testServerInformation{})(db)(router, l)
		p := NewProcessor(l, context.Background(), db)

		seedTemplate(t, db, "GMS", 83, 1, nil)
		id := seedTenant(t, db, p, "GMS", 83, 1, func(rm *RestModel) {
			rm.UsesPin = false
		})

		rr := doReset(t, router, id.String(), `{"data":{"type":"tenants","attributes":{"sections":[]}}}`)
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200, body=%s", rr.Code, rr.Body.String())
		}
		doc := decodeResetResponse(t, rr)
		assertComputedAttributesPresent(t, doc.Data.Attributes)

		var templateDrift bool
		if err := json.Unmarshal(doc.Data.Attributes["templateDrift"], &templateDrift); err != nil {
			t.Fatalf("unmarshal templateDrift: %v", err)
		}
		if templateDrift {
			t.Error("expected templateDrift false")
		}
		var sectionDrift map[string]bool
		if err := json.Unmarshal(doc.Data.Attributes["sectionDrift"], &sectionDrift); err != nil {
			t.Fatalf("unmarshal sectionDrift: %v", err)
		}
		assertAllSectionsFalse(t, sectionDrift)
	})

	t.Run("ScopedSections", func(t *testing.T) {
		db := setupViewTestDB(t)
		l := testLogger()
		router := mux.NewRouter()
		InitResource(testServerInformation{})(db)(router, l)
		p := NewProcessor(l, context.Background(), db)

		seedTemplate(t, db, "GMS", 83, 1, nil)
		id := seedTenant(t, db, p, "GMS", 83, 1, func(rm *RestModel) {
			rm.UsesPin = false
			rm.NPCs = []npcs.RestModel{{NPCId: 9000, Impl: "shop"}}
		})

		rr := doReset(t, router, id.String(), `{"data":{"type":"tenants","attributes":{"sections":["properties"]}}}`)
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200, body=%s", rr.Code, rr.Body.String())
		}
		doc := decodeResetResponse(t, rr)
		assertComputedAttributesPresent(t, doc.Data.Attributes)

		var templateDrift bool
		if err := json.Unmarshal(doc.Data.Attributes["templateDrift"], &templateDrift); err != nil {
			t.Fatalf("unmarshal templateDrift: %v", err)
		}
		if !templateDrift {
			t.Error("expected templateDrift true (npcs still drifted)")
		}
		var sectionDrift map[string]bool
		if err := json.Unmarshal(doc.Data.Attributes["sectionDrift"], &sectionDrift); err != nil {
			t.Fatalf("unmarshal sectionDrift: %v", err)
		}
		if sectionDrift["properties"] {
			t.Error("expected sectionDrift[properties] false")
		}
		if !sectionDrift["npcs"] {
			t.Error("expected sectionDrift[npcs] true")
		}
	})

	t.Run("UnknownSectionIs400", func(t *testing.T) {
		db := setupViewTestDB(t)
		l := testLogger()
		router := mux.NewRouter()
		InitResource(testServerInformation{})(db)(router, l)
		p := NewProcessor(l, context.Background(), db)

		id := seedTenant(t, db, p, "GMS", 83, 1, nil)

		rr := doReset(t, router, id.String(), `{"data":{"type":"tenants","attributes":{"sections":["worlds"]}}}`)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400, body=%s", rr.Code, rr.Body.String())
		}
		assertJSONAPIErrorDocument(t, rr, "400")
	})

	t.Run("UsesPinAliasIs400", func(t *testing.T) {
		db := setupViewTestDB(t)
		l := testLogger()
		router := mux.NewRouter()
		InitResource(testServerInformation{})(db)(router, l)
		p := NewProcessor(l, context.Background(), db)

		id := seedTenant(t, db, p, "GMS", 83, 1, nil)

		rr := doReset(t, router, id.String(), `{"data":{"type":"tenants","attributes":{"sections":["usesPin"]}}}`)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400, body=%s", rr.Code, rr.Body.String())
		}
		assertJSONAPIErrorDocument(t, rr, "400")
	})

	t.Run("MalformedBodyIs400", func(t *testing.T) {
		db := setupViewTestDB(t)
		l := testLogger()
		router := mux.NewRouter()
		InitResource(testServerInformation{})(db)(router, l)
		p := NewProcessor(l, context.Background(), db)

		id := seedTenant(t, db, p, "GMS", 83, 1, nil)

		rr := doReset(t, router, id.String(), `{not json`)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400, body=%s", rr.Code, rr.Body.String())
		}
		assertJSONAPIErrorDocument(t, rr, "400")
	})

	t.Run("UnknownTenantIs404", func(t *testing.T) {
		db := setupViewTestDB(t)
		l := testLogger()
		router := mux.NewRouter()
		InitResource(testServerInformation{})(db)(router, l)

		rr := doReset(t, router, uuid.New().String(), "")
		if rr.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404, body=%s", rr.Code, rr.Body.String())
		}
		assertJSONAPIErrorDocument(t, rr, "404")
	})

	t.Run("NoBaselineIs409", func(t *testing.T) {
		db := setupViewTestDB(t)
		l := testLogger()
		router := mux.NewRouter()
		InitResource(testServerInformation{})(db)(router, l)
		p := NewProcessor(l, context.Background(), db)

		id := seedTenant(t, db, p, "GMS", 99, 9, nil)

		rr := doReset(t, router, id.String(), "")
		if rr.Code != http.StatusConflict {
			t.Fatalf("status = %d, want 409, body=%s", rr.Code, rr.Body.String())
		}
		assertJSONAPIErrorDocument(t, rr, "409")
	})

	t.Run("ValidationFailureIs422", func(t *testing.T) {
		db := setupViewTestDB(t)
		l := testLogger()
		router := mux.NewRouter()
		InitResource(testServerInformation{})(db)(router, l)
		p := NewProcessor(l, context.Background(), db)

		// Same conflicting-unsupported-state fixture as
		// reset_test.go:TestResetById_ValidationFailureIsNotPersisted: a
		// handler that is both live and marked unsupported fails pure
		// socket validation, which ResetById surfaces as a
		// *validationFailureError -- 422 on the reset path, not 400
		// (resource.go's default arm; see resource.go's inline comment on
		// why reset differs from PATCH).
		seedTemplate(t, db, "GMS", 83, 1, func(rm *templates.RestModel) {
			rm.Socket.Handlers = []tmplhandler.RestModel{
				{OpCode: "0x01", Validator: "NoOpValidator", Handler: "LoginHandle", Services: []string{"login"}},
			}
			rm.Socket.Writers = []tmplwriter.RestModel{
				{OpCode: "0x00", Writer: "AuthSuccess", Services: []string{"login"}},
			}
			rm.Socket.Unsupported.Handlers = []string{"LoginHandle"}
		})
		id := seedTenant(t, db, p, "GMS", 83, 1, func(rm *RestModel) {
			rm.UsesPin = false
		})

		rr := doReset(t, router, id.String(), "")
		if rr.Code != http.StatusUnprocessableEntity {
			t.Fatalf("status = %d, want 422, body=%s", rr.Code, rr.Body.String())
		}
		if ct := rr.Header().Get("Content-Type"); ct != "application/vnd.api+json" {
			t.Errorf("Content-Type = %q, want application/vnd.api+json", ct)
		}
		var doc struct {
			Errors []struct {
				Status string         `json:"status"`
				Title  string         `json:"title"`
				Detail string         `json:"detail"`
				Meta   map[string]any `json:"meta"`
			} `json:"errors"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &doc); err != nil {
			t.Fatalf("unmarshal error document: %v (body=%s)", err, rr.Body.String())
		}
		if len(doc.Errors) != 1 {
			t.Fatalf("len(errors) = %d, want 1 (body=%s)", len(doc.Errors), rr.Body.String())
		}
		want := `"LoginHandle" is marked unsupported but is also defined in socket.handlers`
		if doc.Errors[0].Detail != want {
			t.Errorf("errors[0].detail = %q, want %q", doc.Errors[0].Detail, want)
		}
		if doc.Errors[0].Title != "validation failed" {
			t.Errorf("errors[0].title = %q, want %q", doc.Errors[0].Title, "validation failed")
		}
		if got := doc.Errors[0].Meta["path"]; got != "socket.unsupported.handlers[0]" {
			t.Errorf("errors[0].meta.path = %v, want socket.unsupported.handlers[0]", got)
		}
	})

	t.Run("InvalidUUIDIs400", func(t *testing.T) {
		db := setupViewTestDB(t)
		l := testLogger()
		router := mux.NewRouter()
		InitResource(testServerInformation{})(db)(router, l)

		rr := doReset(t, router, "not-a-uuid", "")
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400, body=%s", rr.Code, rr.Body.String())
		}
	})
}

func TestGetTenantCarriesComputedAttributes(t *testing.T) {
	db := setupViewTestDB(t)
	l := testLogger()
	router := mux.NewRouter()
	InitResource(testServerInformation{})(db)(router, l)
	p := NewProcessor(l, context.Background(), db)

	seedTemplate(t, db, "GMS", 83, 1, nil)
	id := seedTenant(t, db, p, "GMS", 83, 1, nil)

	rr := doGetConfigTenants(t, router, "/configurations/tenants/"+id.String())
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rr.Code, rr.Body.String())
	}
	doc := decodeResetResponse(t, rr)
	assertComputedAttributesPresent(t, doc.Data.Attributes)
}

func TestGetTenantsListCarriesComputedAttributes(t *testing.T) {
	db := setupViewTestDB(t)
	l := testLogger()
	router := mux.NewRouter()
	InitResource(testServerInformation{})(db)(router, l)
	p := NewProcessor(l, context.Background(), db)

	seedTemplate(t, db, "GMS", 83, 1, nil)
	seedTenant(t, db, p, "GMS", 83, 1, nil)

	rr := doGetConfigTenants(t, router, "/configurations/tenants")
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rr.Code, rr.Body.String())
	}
	var doc struct {
		Data []struct {
			Attributes map[string]json.RawMessage `json:"attributes"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &doc); err != nil {
		t.Fatalf("unmarshal response document: %v (body=%s)", err, rr.Body.String())
	}
	if len(doc.Data) == 0 {
		t.Fatal("expected at least one tenant in the list")
	}
	assertComputedAttributesPresent(t, doc.Data[0].Attributes)
}

func TestCreateTenantReturnsViewModel(t *testing.T) {
	db := setupViewTestDB(t)
	l := testLogger()
	router := mux.NewRouter()
	InitResource(testServerInformation{})(db)(router, l)

	templateId := seedTemplate(t, db, "GMS", 83, 1, nil)
	tmplRow, err := templates.NewProcessor(l, context.Background(), db).GetById(templateId)
	if err != nil {
		t.Fatalf("failed to load seeded template: %v", err)
	}
	tenantRM := tenantFromTemplate(t, tmplRow)
	attributes, err := json.Marshal(tenantRM)
	if err != nil {
		t.Fatalf("marshal tenant attributes: %v", err)
	}
	body, err := json.Marshal(map[string]any{
		"data": map[string]any{
			"type":       "tenants",
			"attributes": json.RawMessage(attributes),
		},
	})
	if err != nil {
		t.Fatalf("marshal tenant body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/configurations/tenants", strings.NewReader(string(body)))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201, body=%s", rr.Code, rr.Body.String())
	}
	doc := decodeResetResponse(t, rr)
	assertComputedAttributesPresent(t, doc.Data.Attributes)

	var templateDrift bool
	if err := json.Unmarshal(doc.Data.Attributes["templateDrift"], &templateDrift); err != nil {
		t.Fatalf("unmarshal templateDrift: %v", err)
	}
	if templateDrift {
		t.Error("expected templateDrift false when the body was cloned from the template")
	}
}

func TestPatchIgnoresComputedAttributes(t *testing.T) {
	db := setupViewTestDB(t)
	l := testLogger()
	router := mux.NewRouter()
	InitResource(testServerInformation{})(db)(router, l)
	p := NewProcessor(l, context.Background(), db)

	id := seedTenant(t, db, p, "GMS", 83, 1, nil)

	body := `{"data":{"type":"tenants","attributes":{"region":"GMS","majorVersion":83,"minorVersion":1,"usesPin":false,"templateDrift":true,"storedRevision":"deadbeef","sectionDrift":{"socket":true}}}}`
	req := httptest.NewRequest(http.MethodPatch, "/configurations/tenants/"+id.String(), strings.NewReader(body))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204, body=%s", rr.Code, rr.Body.String())
	}

	stored := storedEntity(t, db, id)
	raw := string(stored.Data)
	for _, phantom := range []string{"baselineTemplateId", "baselineRevision", "storedRevision", "templateDrift", "sectionDrift"} {
		if strings.Contains(raw, phantom) {
			t.Errorf("stored Entity.Data contains %q, want none of the computed keys persisted", phantom)
		}
	}

	rr = doGetConfigTenants(t, router, "/configurations/tenants/"+id.String())
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rr.Code, rr.Body.String())
	}
	doc := decodeResetResponse(t, rr)
	var templateDrift bool
	if err := json.Unmarshal(doc.Data.Attributes["templateDrift"], &templateDrift); err != nil {
		t.Fatalf("unmarshal templateDrift: %v", err)
	}
	if templateDrift {
		t.Error("expected no phantom drift after PATCH")
	}
}
