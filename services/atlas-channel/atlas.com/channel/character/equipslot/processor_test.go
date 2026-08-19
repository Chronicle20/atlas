package equipslot

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"
)

// activeExtensionsResponse is a JSON:API equip-slot-extensions collection
// response, mirroring what atlas-character's equipslot resource marshals
// (resource type "equip-slot-extensions").
const activeExtensionsResponse = `{
  "data": [
    { "type": "equip-slot-extensions", "id": "1", "attributes": { "characterId": 9001, "slotIndex": 3, "expiresAt": "2026-01-01T00:00:00Z" } }
  ]
}`

// TestGetActive_DecodesActiveExtensions stands up an httptest server
// emulating atlas-character's equip-slot-extensions lookup and asserts the
// processor decodes the collection and hits the expected character path.
func TestGetActive_DecodesActiveExtensions(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(activeExtensionsResponse))
	}))
	defer srv.Close()

	t.Setenv("CHARACTERS_SERVICE_URL", srv.URL+"/")

	rms, err := NewProcessor(logrus.New(), context.Background()).GetActive(9001)
	if err != nil {
		t.Fatalf("GetActive returned error: %v", err)
	}

	if gotPath != "/characters/9001/equip-slot-extensions" {
		t.Errorf("request path: want /characters/9001/equip-slot-extensions, got %q", gotPath)
	}
	if len(rms) != 1 {
		t.Fatalf("GetActive: got %d extensions, want 1", len(rms))
	}
	if rms[0].CharacterId != 9001 {
		t.Errorf("CharacterId: got %d, want 9001", rms[0].CharacterId)
	}
	if rms[0].SlotIndex != 3 {
		t.Errorf("SlotIndex: got %d, want 3", rms[0].SlotIndex)
	}
	if rms[0].ExpiresAt.IsZero() {
		t.Error("ExpiresAt: want non-zero, got zero")
	}
}
