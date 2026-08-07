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
