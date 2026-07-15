package route_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"atlas-channel/transport/route"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
)

// routeDoc renders a JSON:API document for routes [from, to]. Each route
// gets a distinct uuid derived from its index so the test can assert
// presence of a page-2-only item.
func routeDoc(from, to int, total, number, size, last int) string {
	var b strings.Builder
	for i := from; i <= to; i++ {
		if b.Len() > 0 {
			b.WriteString(",")
		}
		id := fmt.Sprintf("00000000-0000-0000-0000-%012d", i)
		b.WriteString(fmt.Sprintf(
			`{"id":"%s","type":"routes","attributes":{"name":"route-%d","startMapId":100000000,"stagingMapId":100000001,"enRouteMapIds":[100000002],"destinationMapId":200000100,"state":"open_entry","cycleInterval":1800000000000}}`,
			id, i,
		))
	}
	return fmt.Sprintf(
		`{"data":[%s],"meta":{"total":%d,"page":{"number":%d,"size":%d,"last":%d}}}`,
		b.String(), total, number, size, last,
	)
}

// TestInTenantProviderDrainsBeyondOnePage proves
// route.Processor.InTenantProvider (via requests.DrainProvider) fetches
// every page of the tenant's transport routes rather than stopping after
// the first response. atlas-transports' GET /transports/routes is now
// paginated (task-117); IsBoatInMap scans every route in the tenant, a
// genuine semantic-all consumer. The fixture server serves 260 routes
// across two pages of 250 - only a genuine drain picks up the route on
// page 2.
func TestInTenantProviderDrainsBeyondOnePage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		number, _ := strconv.Atoi(r.URL.Query().Get("page[number]"))
		w.Header().Set("Content-Type", "application/vnd.api+json")
		if number == 2 {
			_, _ = w.Write([]byte(routeDoc(251, 260, 260, 2, 250, 2)))
			return
		}
		_, _ = w.Write([]byte(routeDoc(1, 250, 260, 1, 250, 2)))
	}))
	defer srv.Close()
	t.Setenv("ROUTES_SERVICE_URL", srv.URL+"/")

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), ten)
	l, _ := test.NewNullLogger()

	routes, err := route.NewProcessor(l, ctx).GetInTenant()
	if err != nil {
		t.Fatal(err)
	}
	if len(routes) != 260 {
		t.Fatalf("expected 260 routes (full drain), got %d; a single-fetch implementation would return 250", len(routes))
	}

	foundLast := false
	for _, r := range routes {
		if r.Name() == "route-260" {
			foundLast = true
		}
	}
	if !foundLast {
		t.Fatal("route 260 (page 2) must be present; single-fetch impl would miss it")
	}
}
