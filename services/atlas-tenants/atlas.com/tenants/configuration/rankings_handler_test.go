package configuration_test

import (
	"atlas-tenants/configuration"
	"atlas-tenants/test"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	logtest "github.com/sirupsen/logrus/hooks/test"

	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
)

// rankingsHandlerTestDB builds the same in-memory sqlite database
// resource_paginate_test.go uses, plus the outbox_entries table:
// CreateRankingsAndEmit/UpdateRankingsAndEmit/DeleteRankingsAndEmit wrap
// their write in database.ExecuteTransaction and enqueue the status event
// via outbox.EmitProvider, which persists a row to that table inside the
// same transaction (see services/atlas-guilds/.../thread/processor_outbox_test.go
// for the identical pattern in another service).
func rankingsHandlerTestDB(t *testing.T) *httptest.Server {
	t.Helper()
	db := test.SetupTestDB(t)
	if err := outbox.Migration(db); err != nil {
		t.Fatalf("outbox migration: %v", err)
	}
	logger, _ := logtest.NewNullLogger()

	router := mux.NewRouter()
	configuration.RegisterRoutes(db)(testServerInformation{})(router, logger)
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)
	t.Cleanup(func() { test.CleanupTestDB(db) })
	return srv
}

func doRankingsRequest(t *testing.T, method, url string, body []byte) *http.Response {
	t.Helper()
	var req *http.Request
	var err error
	if body != nil {
		req, err = http.NewRequest(method, url, bytes.NewBuffer(body))
	} else {
		req, err = http.NewRequest(method, url, nil)
	}
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	return resp
}

// TestRankingsHandlerWireRoundTrip drives the rankings configuration
// resource through its actual HTTP handlers (CreateRankingsHandler,
// GetRankingsHandler) and the real JSON:API codec (jsonapi.Unmarshal on
// decode via server.ParseInput, jsonapi.MarshalToStruct on encode via
// server.MarshalResponse) — not the processor or TransformRankings /
// ExtractRankings called directly.
//
// It POSTs recomputeIntervalMinutes=17 (a distinctive, non-zero, non-default
// value) and then asserts the literal JSON attribute name
// "recomputeIntervalMinutes" appears in the raw response bytes carrying that
// value. Decoding the response into a struct whose json tag is hardcoded
// here (not reused from configuration.RankingsRestModel) means a json-tag
// typo in the production RestModel — the exact bug class this test guards
// against — shows up as a decode-miss (zero value) or a missing literal
// substring, not a silent pass.
func TestRankingsHandlerWireRoundTrip(t *testing.T) {
	srv := rankingsHandlerTestDB(t)
	tenantId := uuid.New()
	rankingsURL := fmt.Sprintf("%s/tenants/%s/configurations/rankings", srv.URL, tenantId)

	// 1. POST a rankings config with recomputeIntervalMinutes=17 through
	// CreateRankingsHandler + the real JSON:API decode path.
	envelope := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "rankings",
			"attributes": map[string]interface{}{
				"recomputeIntervalMinutes": 17,
			},
		},
	}
	postBody, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("marshal post body: %v", err)
	}

	postResp := doRankingsRequest(t, http.MethodPost, rankingsURL, postBody)
	defer postResp.Body.Close()
	if postResp.StatusCode != http.StatusCreated {
		t.Fatalf("create status = %d, want 201", postResp.StatusCode)
	}

	var postDoc struct {
		Data struct {
			Type       string `json:"type"`
			Id         string `json:"id"`
			Attributes struct {
				RecomputeIntervalMinutes uint32 `json:"recomputeIntervalMinutes"`
			} `json:"attributes"`
		} `json:"data"`
	}
	postRaw := new(bytes.Buffer)
	if _, err := postRaw.ReadFrom(postResp.Body); err != nil {
		t.Fatalf("read create body: %v", err)
	}
	if err := json.Unmarshal(postRaw.Bytes(), &postDoc); err != nil {
		t.Fatalf("decode create body: %v, body=%s", err, postRaw.String())
	}

	// 3. Confirm the resource type string is "rankings".
	if postDoc.Data.Type != "rankings" {
		t.Fatalf("create data.type = %q, want %q", postDoc.Data.Type, "rankings")
	}
	if postDoc.Data.Id == "" {
		t.Fatal("create did not assign an id")
	}
	if postDoc.Data.Attributes.RecomputeIntervalMinutes != 17 {
		t.Fatalf("create data.attributes.recomputeIntervalMinutes = %d, want 17", postDoc.Data.Attributes.RecomputeIntervalMinutes)
	}
	if !strings.Contains(postRaw.String(), `"recomputeIntervalMinutes":17`) {
		t.Fatalf("create response is missing the literal attribute name/value; body=%s", postRaw.String())
	}

	// 2. GET it back through GetRankingsHandler and assert the response
	// body is the single-object {"data":{"type":"rankings","attributes":
	// {"recomputeIntervalMinutes":17}}} shape.
	getResp := doRankingsRequest(t, http.MethodGet, rankingsURL, nil)
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("get status = %d, want 200", getResp.StatusCode)
	}

	getRaw := new(bytes.Buffer)
	if _, err := getRaw.ReadFrom(getResp.Body); err != nil {
		t.Fatalf("read get body: %v", err)
	}

	var getDoc struct {
		Data struct {
			Type       string `json:"type"`
			Attributes struct {
				RecomputeIntervalMinutes uint32 `json:"recomputeIntervalMinutes"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := json.Unmarshal(getRaw.Bytes(), &getDoc); err != nil {
		t.Fatalf("decode get body: %v, body=%s", err, getRaw.String())
	}

	if getDoc.Data.Type != "rankings" {
		t.Fatalf("get data.type = %q, want %q", getDoc.Data.Type, "rankings")
	}
	if getDoc.Data.Attributes.RecomputeIntervalMinutes != 17 {
		t.Fatalf("get data.attributes.recomputeIntervalMinutes = %d, want 17", getDoc.Data.Attributes.RecomputeIntervalMinutes)
	}
	if !strings.Contains(getRaw.String(), `"recomputeIntervalMinutes":17`) {
		t.Fatalf("get response is missing the literal attribute name/value; body=%s", getRaw.String())
	}
	if !strings.Contains(getRaw.String(), `"type":"rankings"`) {
		t.Fatalf("get response is missing the literal resource type; body=%s", getRaw.String())
	}
}
