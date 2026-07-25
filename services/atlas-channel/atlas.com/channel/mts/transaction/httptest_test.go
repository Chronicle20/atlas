package transaction

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"
)

// transactionsResponse is a real JSON:API "transactions" list response mirroring
// what atlas-mts's transaction handler marshals. Serving it verbatim drives the
// same api2go unmarshal path (and the SetToOneReferenceID/SetToManyReferenceIDs
// relationship stubs) the live client uses — a FakeClient mock would bypass it.
const transactionsResponse = `{
  "data": [
    { "type": "transactions", "id": "33333333-3333-3333-3333-333333333333", "attributes": {
        "worldId": 1, "characterId": 100100, "counterpartyId": 200200,
        "itemId": 1302000, "quantity": 3, "totalPrice": 5000, "kind": "purchase",
        "createdAt": "2026-06-17T10:30:00Z" } }
  ]
}`

// TestGetByCharacter_UnmarshalsTransaction stands up an httptest server emulating
// atlas-mts's transaction-history endpoint and asserts the Processor unmarshals the
// JSON:API body into a populated Model. MTS_SERVICE_URL points the RootUrl("MTS")
// client at the server.
func TestGetByCharacter_UnmarshalsTransaction(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(transactionsResponse))
	}))
	defer srv.Close()

	t.Setenv("MTS_SERVICE_URL", srv.URL+"/")

	ms, err := NewProcessor(logrus.New(), context.Background()).GetByCharacter(100100)
	if err != nil {
		t.Fatalf("GetByCharacter returned error: %v", err)
	}

	if gotPath != "/characters/100100/mts/transactions" {
		t.Errorf("request path: want /characters/100100/mts/transactions, got %q", gotPath)
	}
	if len(ms) != 1 {
		t.Fatalf("transactions: want 1, got %d", len(ms))
	}
	m := ms[0]
	if m.Id() != "33333333-3333-3333-3333-333333333333" {
		t.Errorf("id = %q", m.Id())
	}
	if m.TotalPrice() != 5000 {
		t.Errorf("totalPrice = %d, want 5000", m.TotalPrice())
	}
	if m.ItemId() != 1302000 {
		t.Errorf("itemId = %d, want 1302000", m.ItemId())
	}
	if m.Quantity() != 3 {
		t.Errorf("quantity = %d, want 3", m.Quantity())
	}
	if m.Kind() != "purchase" {
		t.Errorf("kind = %q, want purchase", m.Kind())
	}
	if m.CreatedAt().IsZero() {
		t.Errorf("createdAt = zero, want the unmarshaled 2026-06-17T10:30:00Z")
	}
}
