package config_test

import (
	"atlas-transports/transport/config"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// routeDoc renders a JSON:API document for routes [from, to]. Each route's
// name is unique ("Route-<n>") so the test can assert presence of a
// page-2-only item - ExtractRoute mints a fresh uuid.New() id per route
// (ignores the wire "id"), so identity has to be asserted by name instead
// of id.
func routeDoc(from, to int, total, number, size, last int) string {
	var b strings.Builder
	for id := from; id <= to; id++ {
		if b.Len() > 0 {
			b.WriteString(",")
		}
		b.WriteString(fmt.Sprintf(
			`{"id":"%d","type":"routes","attributes":{"name":"Route-%d","startMapId":100000000,"stagingMapId":100000001,"enRouteMapIds":[100000002],"destinationMapId":200000100,"observationMapId":0,"boardingWindowDuration":5,"preDepartureDuration":2,"travelDuration":10,"cycleInterval":30}}`,
			id, id,
		))
	}
	return fmt.Sprintf(
		`{"data":[%s],"meta":{"total":%d,"page":{"number":%d,"size":%d,"last":%d}}}`,
		b.String(), total, number, size, last,
	)
}

// vesselDoc renders a JSON:API document for vessels [from, to]. Unlike
// ExtractRoute, ExtractVessel preserves the wire id, so identity is
// asserted by id.
func vesselDoc(from, to int, total, number, size, last int) string {
	var b strings.Builder
	for id := from; id <= to; id++ {
		if b.Len() > 0 {
			b.WriteString(",")
		}
		b.WriteString(fmt.Sprintf(
			`{"id":"vessel-%d","type":"vessels","attributes":{"name":"Vessel-%d","routeAID":"route-a","routeBID":"route-b","turnaroundDelay":30}}`,
			id, id,
		))
	}
	return fmt.Sprintf(
		`{"data":[%s],"meta":{"total":%d,"page":{"number":%d,"size":%d,"last":%d}}}`,
		b.String(), total, number, size, last,
	)
}

// TestGetRoutesDrainsBeyondOnePage proves config.Processor.GetRoutes (via
// requests.DrainProvider) fetches every page of a tenant's routes rather
// than stopping after the first response. atlas-tenants' GET
// /tenants/{tenantId}/configurations/routes is now paginated (task-117);
// LoadConfigurationsForTenant is a genuine semantic-all startup consumer.
// The fixture server serves 260 routes across two pages of 250 - only a
// genuine drain picks up the route on page 2.
func TestGetRoutesDrainsBeyondOnePage(t *testing.T) {
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
	t.Setenv("TENANTS_SERVICE_URL", srv.URL+"/")

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), ten)
	l, _ := test.NewNullLogger()

	routes, err := config.NewProcessor(l, ctx).GetRoutes(ten.Id().String())
	if err != nil {
		t.Fatal(err)
	}
	if len(routes) != 260 {
		t.Fatalf("expected 260 routes (full drain), got %d; a single-fetch implementation would return 250", len(routes))
	}

	foundLast := false
	for _, r := range routes {
		if r.Name() == "Route-260" {
			foundLast = true
		}
	}
	if !foundLast {
		t.Fatal("Route-260 (page 2) must be present; single-fetch impl would miss it")
	}
}

// TestGetVesselsDrainsBeyondOnePage is the vessels analog of
// TestGetRoutesDrainsBeyondOnePage above (GET
// /tenants/{tenantId}/configurations/vessels).
func TestGetVesselsDrainsBeyondOnePage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		number, _ := strconv.Atoi(r.URL.Query().Get("page[number]"))
		w.Header().Set("Content-Type", "application/vnd.api+json")
		if number == 2 {
			_, _ = w.Write([]byte(vesselDoc(251, 260, 260, 2, 250, 2)))
			return
		}
		_, _ = w.Write([]byte(vesselDoc(1, 250, 260, 1, 250, 2)))
	}))
	defer srv.Close()
	t.Setenv("TENANTS_SERVICE_URL", srv.URL+"/")

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), ten)
	l, _ := test.NewNullLogger()

	vessels, err := config.NewProcessor(l, ctx).GetVessels(ten.Id().String())
	if err != nil {
		t.Fatal(err)
	}
	if len(vessels) != 260 {
		t.Fatalf("expected 260 vessels (full drain), got %d; a single-fetch implementation would return 250", len(vessels))
	}

	foundLast := false
	for _, v := range vessels {
		if v.Id() == "vessel-260" {
			foundLast = true
		}
	}
	if !foundLast {
		t.Fatal("vessel-260 (page 2) must be present; single-fetch impl would miss it")
	}
}
