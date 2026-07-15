package config_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"atlas-transports/instance/config"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
)

// instanceRouteDoc renders a JSON:API document for instance routes [from,
// to]. Each route's name is unique ("InstanceRoute-<n>") so the test can
// assert presence of a page-2-only item - ExtractRoute mints a fresh
// uuid.New() id per route (ignores the wire "id"), so identity has to be
// asserted by name instead of id.
func instanceRouteDoc(from, to int, total, number, size, last int) string {
	var b strings.Builder
	for id := from; id <= to; id++ {
		if b.Len() > 0 {
			b.WriteString(",")
		}
		b.WriteString(fmt.Sprintf(
			`{"id":"%d","type":"instance-routes","attributes":{"name":"InstanceRoute-%d","startMapId":100000000,"transitMapIds":[100000100],"destinationMapId":100000200,"capacity":3,"boardingWindowSeconds":10,"travelDurationSeconds":30,"transitMessage":""}}`,
			id, id,
		))
	}
	return fmt.Sprintf(
		`{"data":[%s],"meta":{"total":%d,"page":{"number":%d,"size":%d,"last":%d}}}`,
		b.String(), total, number, size, last,
	)
}

// TestGetInstanceRoutesDrainsBeyondOnePage proves
// config.Processor.GetInstanceRoutes (via requests.DrainProvider) fetches
// every page of a tenant's instance routes rather than stopping after the
// first response. atlas-tenants' GET
// /tenants/{tenantId}/configurations/instance-routes is now paginated
// (task-117); LoadConfigurationsForTenant is a genuine semantic-all startup
// consumer. The fixture server serves 260 routes across two pages of 250 -
// only a genuine drain picks up the route on page 2.
func TestGetInstanceRoutesDrainsBeyondOnePage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		number, _ := strconv.Atoi(r.URL.Query().Get("page[number]"))
		w.Header().Set("Content-Type", "application/vnd.api+json")
		if number == 2 {
			_, _ = w.Write([]byte(instanceRouteDoc(251, 260, 260, 2, 250, 2)))
			return
		}
		_, _ = w.Write([]byte(instanceRouteDoc(1, 250, 260, 1, 250, 2)))
	}))
	defer srv.Close()
	t.Setenv("TENANTS_SERVICE_URL", srv.URL+"/")

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), ten)
	l, _ := test.NewNullLogger()

	routes, err := config.NewProcessor(l, ctx).GetInstanceRoutes(ten.Id().String())
	if err != nil {
		t.Fatal(err)
	}
	if len(routes) != 260 {
		t.Fatalf("expected 260 routes (full drain), got %d; a single-fetch implementation would return 250", len(routes))
	}

	foundLast := false
	for _, r := range routes {
		if r.Name() == "InstanceRoute-260" {
			foundLast = true
		}
	}
	if !foundLast {
		t.Fatal("InstanceRoute-260 (page 2) must be present; single-fetch impl would miss it")
	}
}
