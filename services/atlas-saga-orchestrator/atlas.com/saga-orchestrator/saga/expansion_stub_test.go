package saga

// expansion_stub_test.go — the stubbed inventory compartment service the
// storage and cash-shop expansion tests read through. It lived in
// trade_expansion_test.go until escrow-at-staging removed the trade expanders'
// need for any inventory read at all (design §5A.7); the remaining users are
// unrelated, so it moved here rather than into one of them.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

// testAsset is one row of a stubbed inventory compartment: the fields
// the storage and cash-shop expanders read off compartment.AssetRestModel.
type testAsset struct {
	CharacterId   uint32
	InventoryType byte
	Slot          int16
	TemplateId    uint32
	Quantity      uint32
	Id            string
}

// testProcessorWithCompartments stands up an httptest inventory service that
// answers GET characters/{id}/inventory/compartments?type={t} from the supplied
// asset table, points INVENTORY_SERVICE_URL at it, and returns a
// *ProcessorImpl wired with a tenant context. The expansion functions reach
// inventory only through compartment.RequestCompartment, so no other processor
// dependency is needed.
func testProcessorWithCompartments(t *testing.T, assets []testAsset) *ProcessorImpl {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		characterId, err := characterIdFromCompartmentPath(r.URL.Path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		invType, err := strconv.ParseUint(r.URL.Query().Get("type"), 10, 8)
		if err != nil {
			http.Error(w, "bad type: "+err.Error(), http.StatusBadRequest)
			return
		}
		var matched []testAsset
		for _, a := range assets {
			if a.CharacterId == characterId && a.InventoryType == byte(invType) {
				matched = append(matched, a)
			}
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(compartmentDoc(byte(invType), matched)))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("INVENTORY_SERVICE_URL", srv.URL+"/")

	// The base URL is resolved per request, not at construction, so the shared
	// constructor can be reused verbatim once the env var is set.
	return newTestExpansionProcessor(t)
}

// characterIdFromCompartmentPath pulls {id} out of
// /characters/{id}/inventory/compartments.
func characterIdFromCompartmentPath(path string) (uint32, error) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 2 || parts[0] != "characters" {
		return 0, fmt.Errorf("unexpected compartment path %q", path)
	}
	id, err := strconv.ParseUint(parts[1], 10, 32)
	if err != nil {
		return 0, fmt.Errorf("unexpected character id in %q: %w", path, err)
	}
	return uint32(id), nil
}

// compartmentDoc renders a JSON:API compartment whose assets relationship is
// materialized from the included block, matching compartment.CompartmentRestModel.
func compartmentDoc(invType byte, assets []testAsset) string {
	refs := make([]map[string]any, 0, len(assets))
	included := make([]map[string]any, 0, len(assets))
	for _, a := range assets {
		refs = append(refs, map[string]any{"type": "assets", "id": a.Id})
		included = append(included, map[string]any{
			"type": "assets",
			"id":   a.Id,
			"attributes": map[string]any{
				"slot":       a.Slot,
				"templateId": a.TemplateId,
				"quantity":   a.Quantity,
				"owner":      "Chronicle",
			},
		})
	}
	doc := map[string]any{
		"data": map[string]any{
			"type":          "compartments",
			"id":            fmt.Sprintf("comp-%d", invType),
			"attributes":    map[string]any{"type": invType, "capacity": 24},
			"relationships": map[string]any{"assets": map[string]any{"data": refs}},
		},
		"included": included,
	}
	b, err := json.Marshal(doc)
	if err != nil {
		panic(err)
	}
	return string(b)
}
