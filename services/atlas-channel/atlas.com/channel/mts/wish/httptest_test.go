package wish

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"
)

// wishEntriesResponse is a real JSON:API "wish-entries" list response mirroring
// what atlas-mts's wishlist handler marshals. Serving it verbatim drives the same
// api2go unmarshal path (and the SetToOneReferenceID/SetToManyReferenceIDs
// relationship stubs) the live client uses — a FakeClient mock would bypass it.
const wishEntriesResponse = `{
  "data": [
    { "type": "wish-entries", "id": "44444444-4444-4444-4444-444444444444", "attributes": {
        "worldId": 2, "serial": 7777, "characterId": 9001, "itemId": 1302000,
        "listingSerial": 8888, "price": 1500, "count": 2,
        "createdAt": "2026-06-17T10:30:00Z" } }
  ]
}`

// TestGetByCharacterAndType_UnmarshalsWish stands up an httptest server emulating
// atlas-mts's wishlist endpoint and asserts the Processor unmarshals the JSON:API
// body into a populated Model. MTS_SERVICE_URL points the RootUrl("MTS") client at
// the server.
func TestGetByCharacterAndType_UnmarshalsWish(t *testing.T) {
	var gotPath, gotType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotType = r.URL.Query().Get("type")
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(wishEntriesResponse))
	}))
	defer srv.Close()

	t.Setenv("MTS_SERVICE_URL", srv.URL+"/")

	ms, err := NewProcessor(logrus.New(), context.Background()).GetByCharacterAndType(9001, TypeCart)
	if err != nil {
		t.Fatalf("GetByCharacterAndType returned error: %v", err)
	}

	if gotPath != "/characters/9001/mts/wishlist" {
		t.Errorf("request path: want /characters/9001/mts/wishlist, got %q", gotPath)
	}
	if gotType != TypeCart {
		t.Errorf("type query: want %q, got %q", TypeCart, gotType)
	}
	if len(ms) != 1 {
		t.Fatalf("wish entries: want 1, got %d", len(ms))
	}
	m := ms[0]
	if m.Id() != "44444444-4444-4444-4444-444444444444" {
		t.Errorf("id = %q", m.Id())
	}
	if m.ItemId() != 1302000 {
		t.Errorf("itemId = %d, want 1302000", m.ItemId())
	}
	if m.Price() != 1500 {
		t.Errorf("price = %d, want 1500", m.Price())
	}
	if m.Serial() != 7777 {
		t.Errorf("serial = %d, want 7777", m.Serial())
	}
	if m.CharacterId() != 9001 {
		t.Errorf("characterId = %d, want 9001", m.CharacterId())
	}
}
