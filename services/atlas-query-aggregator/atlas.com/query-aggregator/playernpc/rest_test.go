package playernpc

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/sirupsen/logrus"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// TestGetEligibility_HappyPath asserts GetEligibility issues the eligibility
// GET with characterId/mapId/worldId on the query string and decodes the
// plain (non-JSON:API) response body into an EligibilityModel -- the real
// HTTP/JSON decode path FILE-05's processor.go split left unexercised.
func TestGetEligibility_HappyPath(t *testing.T) {
	const wantCharacterId = uint32(42)
	const wantMapId = _map.Id(100000000)
	const wantWorldId = world.Id(1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("characterId") != strconv.Itoa(int(wantCharacterId)) {
			t.Errorf("characterId = %q, want %q", q.Get("characterId"), strconv.Itoa(int(wantCharacterId)))
		}
		if q.Get("mapId") != strconv.Itoa(int(wantMapId)) {
			t.Errorf("mapId = %q, want %q", q.Get("mapId"), strconv.Itoa(int(wantMapId)))
		}
		if q.Get("worldId") != strconv.Itoa(int(wantWorldId)) {
			t.Errorf("worldId = %q, want %q", q.Get("worldId"), strconv.Itoa(int(wantWorldId)))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"eligible": true, "reason": "no active player npc"}`))
	}))
	defer srv.Close()
	defer SetBaseURLForTest(srv.URL)()

	p := NewProcessor(logrus.New(), context.Background())
	m, err := p.GetEligibility(wantCharacterId, wantMapId, wantWorldId)
	if err != nil {
		t.Fatalf("GetEligibility: %v", err)
	}
	if !m.Eligible() {
		t.Errorf("Eligible() = false, want true")
	}
	if m.Reason() != "no active player npc" {
		t.Errorf("Reason() = %q, want %q", m.Reason(), "no active player npc")
	}
}

// TestGetEligibility_InfrastructureError asserts a non-200 response from the
// eligibility endpoint surfaces as an error rather than a false-eligible
// result, and that the fail-closed NewUnavailableEligibility path a caller
// takes on that error (validation/context.go's GetPlayerNpcEligibility)
// produces an ineligible model with the caller's own reason.
func TestGetEligibility_InfrastructureError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	defer SetBaseURLForTest(srv.URL)()

	p := NewProcessor(logrus.New(), context.Background())
	_, err := p.GetEligibility(42, _map.Id(100000000), world.Id(1))
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	unavailable := NewUnavailableEligibility("player npc eligibility unavailable")
	if unavailable.Eligible() {
		t.Errorf("Eligible() = true, want false")
	}
	if unavailable.Reason() != "player npc eligibility unavailable" {
		t.Errorf("Reason() = %q, want %q", unavailable.Reason(), "player npc eligibility unavailable")
	}
}
