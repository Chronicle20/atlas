package transport_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"atlas-saga-orchestrator/transport"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
)

// instanceRouteDoc renders a JSON:API document for instance routes [from,
// to]. Each route's name is unique ("route-<n>") so the test can assert a
// page-2-only route is found by name.
func instanceRouteDoc(from, to int, total, number, size, last int) string {
	var b strings.Builder
	for i := from; i <= to; i++ {
		if b.Len() > 0 {
			b.WriteString(",")
		}
		id := fmt.Sprintf("00000000-0000-0000-0000-%012d", i)
		b.WriteString(fmt.Sprintf(
			`{"id":"%s","type":"instance-routes","attributes":{"name":"route-%d"}}`,
			id, i,
		))
	}
	return fmt.Sprintf(
		`{"data":[%s],"meta":{"total":%d,"page":{"number":%d,"size":%d,"last":%d}}}`,
		b.String(), total, number, size, last,
	)
}

// TestGetRouteByNameDrainsBeyondOnePage proves GetRouteByName (via
// requests.DrainProvider) fetches every page of the instance routes list
// rather than stopping after the first response. atlas-transports' GET
// /transports/instance-routes is now paginated (task-117); GetRouteByName
// scans every route in the tenant by name, a genuine semantic-all consumer.
// The fixture server serves 260 routes across two pages of 250 - only a
// genuine drain can find a route whose name lives on page 2.
func TestGetRouteByNameDrainsBeyondOnePage(t *testing.T) {
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
	t.Setenv("TRANSPORTS_URL_SERVICE_URL", srv.URL+"/")

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), ten)
	l, _ := test.NewNullLogger()

	route, err := transport.GetRouteByName(l, ctx)("route-260")
	if err != nil {
		t.Fatalf("expected route-260 (page 2) to be found via drain, got error: %v", err)
	}
	if route.Name != "route-260" {
		t.Fatalf("got route name %q, want route-260", route.Name)
	}
}
