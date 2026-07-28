package holding

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"
)

// holdingsResponse is a real JSON:API "holdings" list response mirroring what
// atlas-mts's holding handler marshals. It is served verbatim so the test drives
// the same api2go unmarshal path (and the SetToOneReferenceID/SetToManyReferenceIDs
// relationship stubs) the live client uses — a FakeClient mock would bypass it.
const holdingsResponse = `{
  "data": [
    { "type": "holdings", "id": "11111111-1111-1111-1111-111111111111", "attributes": {
        "worldId": 1, "itcSn": 4242, "ownerId": 100100, "origin": "purchased",
        "templateId": 1302000, "quantity": 3 } }
  ]
}`

// TestGetByCharacter_UnmarshalsHolding stands up an httptest server emulating
// atlas-mts's holding read endpoint and asserts the Processor unmarshals the
// JSON:API body into a populated Model. MTS_SERVICE_URL points the RootUrl("MTS")
// client at the server.
func TestGetByCharacter_UnmarshalsHolding(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(holdingsResponse))
	}))
	defer srv.Close()

	t.Setenv("MTS_SERVICE_URL", srv.URL+"/")

	ms, err := NewProcessor(logrus.New(), context.Background()).GetByCharacter(100100)
	if err != nil {
		t.Fatalf("GetByCharacter returned error: %v", err)
	}

	if gotPath != "/characters/100100/mts/holding" {
		t.Errorf("request path: want /characters/100100/mts/holding, got %q", gotPath)
	}
	if len(ms) != 1 {
		t.Fatalf("holdings: want 1, got %d", len(ms))
	}
	m := ms[0]
	if m.Id() != "11111111-1111-1111-1111-111111111111" {
		t.Errorf("id = %q", m.Id())
	}
	if m.TemplateId() != 1302000 {
		t.Errorf("templateId = %d, want 1302000", m.TemplateId())
	}
	if m.ItcSn() != 4242 {
		t.Errorf("itcSn = %d, want 4242", m.ItcSn())
	}
	if m.Quantity() != 3 {
		t.Errorf("quantity = %d, want 3", m.Quantity())
	}
}
