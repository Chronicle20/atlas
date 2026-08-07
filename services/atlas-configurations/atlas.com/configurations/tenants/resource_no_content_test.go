package tenants

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

// The tenant twin of the template contract: PATCH and DELETE return no
// representation, so they answer 204 rather than net/http's implicit 200 with
// a zero-length body.
func TestUpdateConfigurationTenantReturnsNoContent(t *testing.T) {
	db := setupTestDB(t)
	l := testLogger()

	id := "00000000-0000-0000-0000-0000000004b1"
	input := createTestRestModel("GMS", 83, 1)
	input.Id = id
	if _, err := NewProcessor(l, context.Background(), db).Create(input); err != nil {
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
			"type":       "tenants",
			"id":         id,
			"attributes": json.RawMessage(attributes),
		},
	})
	if err != nil {
		t.Fatalf("body marshal failed: %v", err)
	}

	req, err := http.NewRequest(http.MethodPatch, "/configurations/tenants/"+id, bytes.NewReader(body))
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

func TestDeleteConfigurationTenantReturnsNoContent(t *testing.T) {
	db := setupTestDB(t)
	l := testLogger()

	id := "00000000-0000-0000-0000-0000000004b2"
	input := createTestRestModel("GMS", 83, 1)
	input.Id = id
	if _, err := NewProcessor(l, context.Background(), db).Create(input); err != nil {
		t.Fatalf("seed create failed: %v", err)
	}

	router := mux.NewRouter()
	InitResource(testServerInformation{})(db)(router, l)

	req, err := http.NewRequest(http.MethodDelete, "/configurations/tenants/"+id, nil)
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
