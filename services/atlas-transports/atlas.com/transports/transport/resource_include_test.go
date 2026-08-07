package transport

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	logtest "github.com/sirupsen/logrus/hooks/test"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

// routeListDoc is the subset of the JSON:API document shape this test needs:
// the route's id/attributes/relationship linkage plus the top-level included
// array.
type routeListDoc struct {
	Data []struct {
		ID         string `json:"id"`
		Attributes struct {
			StartMapID uint32 `json:"startMapId"`
		} `json:"attributes"`
		Relationships struct {
			Schedule struct {
				Data []struct {
					ID string `json:"id"`
				} `json:"data"`
			} `json:"schedule"`
		} `json:"relationships"`
	} `json:"data"`
	Included []json.RawMessage `json:"included"`
}

// TestGetAllRoutesScheduleIsOptIn proves the board's payload guarantee: the
// list endpoint does not ship a day's worth of trip rows unless asked.
func TestGetAllRoutesScheduleIsOptIn(t *testing.T) {
	setupTransportTestRegistry(t)
	tm, ctx := newTestTenantContext(t)

	routeId := uuid.MustParse("00000000-0000-0000-0000-0000000004a0")
	trip, err := NewTripScheduleBuilder().
		SetTripId(uuid.New()).
		SetRouteId(routeId).
		SetBoardingOpen(time.Date(2023, 1, 1, 8, 0, 0, 0, time.UTC)).
		SetBoardingClosed(time.Date(2023, 1, 1, 8, 5, 0, 0, time.UTC)).
		SetDeparture(time.Date(2023, 1, 1, 8, 7, 0, 0, time.UTC)).
		SetArrival(time.Date(2023, 1, 1, 8, 17, 0, 0, time.UTC)).
		Build()
	if err != nil {
		t.Fatalf("seed trip build failed: %v", err)
	}

	m, err := NewBuilder("Include Route").
		SetId(routeId).
		SetStartMapId(_map.Id(100000000)).
		SetStagingMapId(_map.Id(100000001)).
		SetEnRouteMapIds([]_map.Id{_map.Id(100000002)}).
		SetDestinationMapId(_map.Id(200000100)).
		SetBoardingWindowDuration(5 * time.Minute).
		SetPreDepartureDuration(2 * time.Minute).
		SetTravelDuration(10 * time.Minute).
		SetCycleInterval(30 * time.Minute).
		SetSchedule([]TripScheduleModel{trip}).
		Build()
	if err != nil {
		t.Fatalf("seed build failed: %v", err)
	}
	getRouteRegistry().AddTenant(ctx, []Model{m})

	logger, _ := logtest.NewNullLogger()
	router := mux.NewRouter()
	InitResource(testServerInformation{})(router, logger)

	read := func(path string) []json.RawMessage {
		rr := doGetRoutes(t, router, tm.Id(), path)
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200, body=%s", rr.Code, rr.Body.String())
		}
		var doc struct {
			Included []json.RawMessage `json:"included"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &doc); err != nil {
			t.Fatalf("unmarshal: %v, body=%s", err, rr.Body.String())
		}
		return doc.Included
	}

	readDoc := func(path string) routeListDoc {
		rr := doGetRoutes(t, router, tm.Id(), path)
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200, body=%s", rr.Code, rr.Body.String())
		}
		var doc routeListDoc
		if err := json.Unmarshal(rr.Body.Bytes(), &doc); err != nil {
			t.Fatalf("unmarshal: %v, body=%s", err, rr.Body.String())
		}
		return doc
	}

	t.Run("DefaultOmitsIncluded", func(t *testing.T) {
		if included := read("/transports/routes"); len(included) != 0 {
			t.Fatalf("len(included) = %d, want 0", len(included))
		}
	})

	t.Run("IncludeScheduleAttachesTrips", func(t *testing.T) {
		if included := read("/transports/routes?include=schedule"); len(included) != 1 {
			t.Fatalf("len(included) = %d, want 1", len(included))
		}
	})

	// The filter[startMapId] branch (resource.go's other Transform call site)
	// selects the same transformer as the all-routes branch above. These two
	// subtests exercise it directly so a regression that hardcoded that
	// branch back to Transform/TransformSummary would fail here even though
	// the all-routes branch stayed correct.
	t.Run("FilteredRouteOmitsIncludedByDefault", func(t *testing.T) {
		doc := readDoc("/transports/routes?filter[startMapId]=100000000")
		if len(doc.Data) != 1 {
			t.Fatalf("len(data) = %d, want 1", len(doc.Data))
		}
		if doc.Data[0].ID != routeId.String() {
			t.Fatalf("data[0].id = %s, want %s", doc.Data[0].ID, routeId.String())
		}
		if doc.Data[0].Attributes.StartMapID != 100000000 {
			t.Fatalf("attributes.startMapId = %d, want 100000000", doc.Data[0].Attributes.StartMapID)
		}
		if len(doc.Data[0].Relationships.Schedule.Data) != 0 {
			t.Fatalf("relationships.schedule.data len = %d, want 0", len(doc.Data[0].Relationships.Schedule.Data))
		}
		if len(doc.Included) != 0 {
			t.Fatalf("len(included) = %d, want 0", len(doc.Included))
		}
	})

	t.Run("FilteredRouteWithIncludeAttachesSchedule", func(t *testing.T) {
		doc := readDoc("/transports/routes?filter[startMapId]=100000000&include=schedule")
		if len(doc.Data) != 1 {
			t.Fatalf("len(data) = %d, want 1", len(doc.Data))
		}
		if doc.Data[0].ID != routeId.String() {
			t.Fatalf("data[0].id = %s, want %s", doc.Data[0].ID, routeId.String())
		}
		if len(doc.Data[0].Relationships.Schedule.Data) != 1 {
			t.Fatalf("relationships.schedule.data len = %d, want 1", len(doc.Data[0].Relationships.Schedule.Data))
		}
		if doc.Data[0].Relationships.Schedule.Data[0].ID != trip.TripId().String() {
			t.Fatalf("relationships.schedule.data[0].id = %s, want %s", doc.Data[0].Relationships.Schedule.Data[0].ID, trip.TripId().String())
		}
		if len(doc.Included) != 1 {
			t.Fatalf("len(included) = %d, want 1", len(doc.Included))
		}
	})

	t.Run("DetailAlwaysIncludesSchedule", func(t *testing.T) {
		rr := doGetRoutes(t, router, tm.Id(), "/transports/routes/"+routeId.String())
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200, body=%s", rr.Code, rr.Body.String())
		}
		var doc struct {
			Included []json.RawMessage `json:"included"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &doc); err != nil {
			t.Fatalf("unmarshal: %v, body=%s", err, rr.Body.String())
		}
		if len(doc.Included) != 1 {
			t.Fatalf("len(included) = %d, want 1", len(doc.Included))
		}
	})
}
