package character

import (
	"atlas-login/inventory"
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// counterValue reads a labeled counter from the default gatherer (0 when the
// series does not exist yet).
func counterValue(t *testing.T, name, labelName, labelValue string) float64 {
	t.Helper()
	mfs, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("gather: %v", err)
	}
	for _, mf := range mfs {
		if mf.GetName() != name {
			continue
		}
		for _, m := range mf.GetMetric() {
			for _, lp := range m.GetLabel() {
				if lp.GetName() == labelName && lp.GetValue() == labelValue {
					return m.GetCounter().GetValue()
				}
			}
		}
	}
	return 0
}

func testContext(t *testing.T) context.Context {
	t.Helper()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	return tenant.WithContext(context.Background(), ten)
}

func TestInventoryDecoratorDegradesLoudly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError) // terminal, not retried
	}))
	defer srv.Close()
	t.Setenv("INVENTORY_SERVICE_URL", srv.URL+"/")

	logger, hook := test.NewNullLogger()
	before := counterValue(t, "atlas_enrichment_degraded_total", "component", "login.character.inventory")

	p := NewProcessor(logger, testContext(t)).(*ProcessorImpl)
	m := Model{id: 42}
	decorated := p.InventoryDecorator()(m)

	if !reflect.DeepEqual(decorated, m) {
		t.Fatalf("expected un-enriched model on failure")
	}
	found := false
	for _, e := range hook.AllEntries() {
		if e.Level == logrus.WarnLevel && strings.Contains(e.Message, "Enrichment degraded") && strings.Contains(e.Message, "42") {
			found = true
		}
	}
	if !found {
		t.Fatal("expected a Warn log naming the degradation and character id 42")
	}
	after := counterValue(t, "atlas_enrichment_degraded_total", "component", "login.character.inventory")
	if after-before != 1 {
		t.Fatalf("expected degradation counter delta 1, got %v", after-before)
	}
}

// Incident replay (PRD acceptance): one transient 503 from atlas-inventory
// must be absorbed by the client retry — the decorated character keeps its
// equipment and nothing degrades.
func TestInventoryDecoratorRetriesThroughTransient503(t *testing.T) {
	rm := inventory.RestModel{Id: uuid.New(), CharacterId: 42}
	body, err := jsonapi.Marshal(rm)
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if attempts.Add(1) == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}))
	defer srv.Close()
	t.Setenv("INVENTORY_SERVICE_URL", srv.URL+"/")

	logger, hook := test.NewNullLogger()
	p := NewProcessor(logger, testContext(t)).(*ProcessorImpl)
	m := Model{id: 42}
	decorated := p.InventoryDecorator()(m)

	if attempts.Load() != 2 {
		t.Fatalf("expected the 503 to be retried (2 attempts), got %d", attempts.Load())
	}
	if reflect.DeepEqual(decorated, m) {
		t.Fatal("expected enriched model after successful retry")
	}
	for _, e := range hook.AllEntries() {
		if strings.Contains(e.Message, "Enrichment degraded") {
			t.Fatalf("nothing may degrade when the retry succeeds: %s", e.Message)
		}
	}
}
