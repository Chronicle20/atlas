package inventory

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func newTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	return tm
}

// TestHasItem_ServedWithIncludedAsset drives a real atlas-inventory compartment
// document (assets to-many relationship + included asset) through the api2go
// SetReferencedStructs include-unmarshal path and asserts HasItem finds the
// matching templateId. Without that path the asset TemplateId/Quantity stay
// zero and the possession check silently returns false (EXT-02). Item 4080000
// (Omok set) classifies to the ETC compartment (type=4).
func TestHasItem_ServedWithIncludedAsset(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		want := "/api/characters/54321/inventory/compartments?type=4&include=assets"
		if got := r.URL.Path + "?" + r.URL.RawQuery; !strings.HasSuffix(got, want) {
			t.Errorf("unexpected request: got %q want suffix %q", got, want)
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{
			"data": {
				"type": "compartments",
				"id": "00000000-0000-0000-0000-000000000001",
				"attributes": { "type": 4, "capacity": 96 },
				"relationships": {
					"assets": { "data": [ { "type": "assets", "id": "7" } ] }
				}
			},
			"included": [ {
				"type": "assets",
				"id": "7",
				"attributes": { "slot": 1, "templateId": 4080000, "quantity": 1 }
			} ]
		}`))
	}))
	defer srv.Close()
	defer SetBaseURLForTest(srv.URL)()

	ctx := tenant.WithContext(context.Background(), newTestTenant(t))
	p := NewProcessor(logrus.New(), ctx)

	has, err := p.HasItem(54321, 4080000)
	if err != nil {
		t.Fatalf("HasItem: %v", err)
	}
	if !has {
		t.Fatalf("HasItem(4080000): want true, got false")
	}

	// An item the compartment does not hold must report false.
	hasOther, err := p.HasItem(54321, 4080001)
	if err != nil {
		t.Fatalf("HasItem(other): %v", err)
	}
	if hasOther {
		t.Fatalf("HasItem(4080001): want false, got true")
	}
}
