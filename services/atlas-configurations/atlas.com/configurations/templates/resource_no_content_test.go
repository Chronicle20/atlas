package templates

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// PATCH and DELETE on a template return no representation. They previously
// fell through to net/http's implicit 200 with a zero-length body, which reads
// as "here is your resource" to any JSON:API client - atlas-ui's api.patch
// resolved to undefined and threw on `response.data` AFTER the write had
// landed. A bodiless success is a 204.
func TestUpdateConfigurationTemplateReturnsNoContent(t *testing.T) {
	db := setupTestDB(t)
	l := testLogger()

	rm := createTestRestModel("GMS", 83, 1)
	seed, err := json.Marshal(rm)
	if err != nil {
		t.Fatalf("seed marshal failed: %v", err)
	}
	id := uuid.MustParse("00000000-0000-0000-0000-0000000004a1")
	if err := db.Create(&Entity{
		Id:           id,
		Region:       rm.Region,
		MajorVersion: rm.MajorVersion,
		MinorVersion: rm.MinorVersion,
		Data:         seed,
	}).Error; err != nil {
		t.Fatalf("seed create failed: %v", err)
	}

	router := mux.NewRouter()
	InitResource(testServerInformation{})(db)(router, l)

	updated := createTestRestModel("GMS", 83, 2)
	attributes, err := json.Marshal(updated)
	if err != nil {
		t.Fatalf("attribute marshal failed: %v", err)
	}
	body, err := json.Marshal(map[string]any{
		"data": map[string]any{
			"type":       "templates",
			"id":         id.String(),
			"attributes": json.RawMessage(attributes),
		},
	})
	if err != nil {
		t.Fatalf("body marshal failed: %v", err)
	}

	req, err := http.NewRequest(http.MethodPatch, "/configurations/templates/"+id.String(), bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204, body=%s", rr.Code, rr.Body.String())
	}
	if rr.Body.Len() != 0 {
		t.Fatalf("body = %q, want empty", rr.Body.String())
	}
}

func TestDeleteConfigurationTemplateReturnsNoContent(t *testing.T) {
	db := setupTestDB(t)
	l := testLogger()

	rm := createTestRestModel("GMS", 83, 1)
	seed, err := json.Marshal(rm)
	if err != nil {
		t.Fatalf("seed marshal failed: %v", err)
	}
	id := uuid.MustParse("00000000-0000-0000-0000-0000000004a2")
	if err := db.Create(&Entity{
		Id:           id,
		Region:       rm.Region,
		MajorVersion: rm.MajorVersion,
		MinorVersion: rm.MinorVersion,
		Data:         seed,
	}).Error; err != nil {
		t.Fatalf("seed create failed: %v", err)
	}

	router := mux.NewRouter()
	InitResource(testServerInformation{})(db)(router, l)

	req, err := http.NewRequest(http.MethodDelete, "/configurations/templates/"+id.String(), nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204, body=%s", rr.Code, rr.Body.String())
	}
	if rr.Body.Len() != 0 {
		t.Fatalf("body = %q, want empty", rr.Body.String())
	}
}
