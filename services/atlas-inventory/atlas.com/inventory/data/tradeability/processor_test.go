package tradeability

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
)

// TestExtractCarriesBothFields proves the shared Extract does not silently drop
// one of the two fields the karma gates need.
func TestExtractCarriesBothFields(t *testing.T) {
	m, err := extract(EquipmentRestModel{Id: 1002357, TradeBlock: true, TradeAvailable: 1})
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if !m.TradeBlock() {
		t.Fatal("TradeBlock() = false, want true")
	}
	if m.TradeAvailable() != 1 {
		t.Fatalf("TradeAvailable() = %d, want 1", m.TradeAvailable())
	}
}

// TestByIdProviderRejectsUnknownCompartment: an unknown compartment must be an
// error the caller refuses on, never a zero-valued permissive default.
func TestByIdProviderRejectsUnknownCompartment(t *testing.T) {
	p := &ProcessorImpl{}
	if _, err := p.ByIdProvider(inventory.Type(99), 1002357)(); err == nil {
		t.Fatal("ByIdProvider(99) returned no error; an unknown compartment must refuse")
	}
}

// compartmentFixture pins one compartment's expected request path and
// JSON:API resource type name against the wire model's own GetName(), so a
// typo in either requests.go's path constants or rest.go's GetName() fails
// the test instead of silently decoding to a zero-valued Model (a runtime 404
// otherwise catches this only in production).
type compartmentFixture struct {
	name       string
	inv        inventory.Type
	templateId item.Id
	wantPath   string
	resource   string
}

var compartmentFixtures = []compartmentFixture{
	{"equip", inventory.TypeValueEquip, 1002357, "/data/equipment/1002357", EquipmentRestModel{}.GetName()},
	{"use", inventory.TypeValueUse, 2000000, "/data/consumables/2000000", ConsumableRestModel{}.GetName()},
	{"setup", inventory.TypeValueSetup, 3010000, "/data/setups/3010000", SetupRestModel{}.GetName()},
	{"etc", inventory.TypeValueETC, 4000000, "/data/etcs/4000000", EtcRestModel{}.GetName()},
	{"cash", inventory.TypeValueCash, 5000000, "/data/cash/items/5000000", CashRestModel{}.GetName()},
}

// TestByIdProvider_AllCompartments stands up an httptest server per
// compartment, serves a JSON:API document whose "type" matches what that
// compartment's RestModel.GetName() returns, and asserts both the request
// path the client actually issued and the decoded TradeBlock/TradeAvailable
// fields. A wrong path constant or a wrong GetName() would either miss the
// handler (path mismatch, surfaced as a non-200) or fail api2go's type check
// (resource mismatch) -- either way this test fails rather than passing
// against a zero-valued decode.
func TestByIdProvider_AllCompartments(t *testing.T) {
	for _, tc := range compartmentFixtures {
		t.Run(tc.name, func(t *testing.T) {
			var gotPath string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				w.Header().Set("Content-Type", "application/vnd.api+json")
				_, _ = fmt.Fprintf(w, `{"data":{"type":%q,"id":"%d","attributes":{"tradeBlock":true,"tradeAvailable":2}}}`,
					tc.resource, tc.templateId)
			}))
			defer srv.Close()

			t.Setenv("DATA_SERVICE_URL", srv.URL+"/")

			p := NewProcessor(logrus.New(), context.Background())
			m, err := p.Get(tc.inv, tc.templateId)
			if err != nil {
				t.Fatalf("Get(%s, %d): %v", tc.name, tc.templateId, err)
			}

			if gotPath != tc.wantPath {
				t.Errorf("request path: want %q, got %q", tc.wantPath, gotPath)
			}
			if !m.TradeBlock() {
				t.Errorf("TradeBlock(): want true, got false")
			}
			if m.TradeAvailable() != 2 {
				t.Errorf("TradeAvailable(): want 2, got %d", m.TradeAvailable())
			}
		})
	}
}
