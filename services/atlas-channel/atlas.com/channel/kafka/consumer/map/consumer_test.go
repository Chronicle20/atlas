package _map

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
)

func newTestCtx(t *testing.T) context.Context {
	t.Helper()
	tn, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	return tenant.WithContext(context.Background(), tn)
}

func newTestField() field.Model {
	return field.NewBuilder(world.Id(1), channel.Id(1), _map.Id(100000000)).SetInstance(uuid.Nil).Build()
}

// characterJSON returns a minimal JSON:API character response for the given ID.
func characterJSON(id uint32) string {
	return fmt.Sprintf(`{
		"data": {
			"type": "characters",
			"id": "%d",
			"attributes": {
				"accountId": 1,
				"worldId": 1,
				"name": "TestChar%d",
				"level": 1,
				"experience": 0,
				"gachaponExperience": 0,
				"strength": 4,
				"dexterity": 4,
				"intelligence": 4,
				"luck": 4,
				"hp": 50,
				"maxHp": 50,
				"mp": 5,
				"maxMp": 5,
				"meso": 0,
				"hpMpUsed": 0,
				"jobId": 0,
				"skinColor": 0,
				"gender": 0,
				"fame": 0,
				"hair": 30000,
				"face": 20000,
				"ap": 0,
				"sp": "0,0,0,0,0,0,0,0,0,0",
				"mapId": 100000000,
				"spawnPoint": 0,
				"gm": 0,
				"x": 0,
				"y": 0,
				"stance": 0
			}
		}
	}`, id, id)
}

// mapCharactersJSON returns a JSON:API list response for the given character IDs.
func mapCharactersJSON(ids ...uint32) string {
	items := make([]string, len(ids))
	for i, id := range ids {
		items[i] = fmt.Sprintf(`{"type":"characters","id":"%d","attributes":{}}`, id)
	}
	return `{"data":[` + strings.Join(items, ",") + `]}`
}

// TestFetchOtherCharactersInMap_SkipsNotFound verifies that a 404 for one
// character is skipped (Warn logged) but the rest are returned successfully.
func TestFetchOtherCharactersInMap_SkipsNotFound(t *testing.T) {
	logger, hook := test.NewNullLogger()
	ctx := newTestCtx(t)
	f := newTestField()

	const selfId uint32 = 1
	const missingId uint32 = 2
	const presentId uint32 = 3

	// Create a single httptest server that handles both MAPS and CHARACTERS URLs.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		switch {
		case strings.Contains(r.URL.Path, "/instances/") && strings.HasSuffix(r.URL.Path, "/characters/"):
			// atlas-maps GET /worlds/{w}/channels/{c}/maps/{m}/instances/{i}/characters/
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, mapCharactersJSON(selfId, missingId, presentId))
		case strings.Contains(r.URL.Path, fmt.Sprintf("/characters/%d", missingId)):
			// atlas-character GET for missingId returns 404
			w.WriteHeader(http.StatusNotFound)
		case strings.Contains(r.URL.Path, fmt.Sprintf("/characters/%d", presentId)):
			// atlas-character GET for presentId returns OK
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, characterJSON(presentId))
		default:
			// any inventory/pet endpoints return empty
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	t.Setenv("MAPS_SERVICE_URL", srv.URL+"/")
	t.Setenv("CHARACTERS_SERVICE_URL", srv.URL+"/")

	cms, err := fetchOtherCharactersInMap(logger, ctx, f, selfId)
	if err != nil {
		t.Fatalf("fetchOtherCharactersInMap returned unexpected error: %v", err)
	}

	// missingId should be absent from results (skipped via Warn)
	if _, ok := cms[missingId]; ok {
		t.Errorf("missingId [%d] should have been skipped but is present in result", missingId)
	}

	// presentId should be present in results
	if _, ok := cms[presentId]; !ok {
		t.Errorf("presentId [%d] should be in result but is absent", presentId)
	}

	// selfId should be excluded (it's the excludeId)
	if _, ok := cms[selfId]; ok {
		t.Errorf("selfId [%d] should be excluded from result", selfId)
	}

	// A Warn log should have been emitted for the missing character
	found := false
	for _, entry := range hook.Entries {
		if strings.Contains(entry.Message, "Skipping stale registry entry") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected a Warn log for the stale/missing character, but none was found")
	}
}

// TestFetchOtherCharactersInMap_InfraErrorIsHardFailure verifies that a
// non-404 error from atlas-character propagates as a hard failure (not skipped).
func TestFetchOtherCharactersInMap_InfraErrorIsHardFailure(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := newTestCtx(t)
	f := newTestField()

	const selfId uint32 = 1
	const badId uint32 = 2

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		switch {
		case strings.Contains(r.URL.Path, "/instances/") && strings.HasSuffix(r.URL.Path, "/characters/"):
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, mapCharactersJSON(selfId, badId))
		default:
			// Return a 500 for all character fetches
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer srv.Close()

	t.Setenv("MAPS_SERVICE_URL", srv.URL+"/")
	t.Setenv("CHARACTERS_SERVICE_URL", srv.URL+"/")

	_, err := fetchOtherCharactersInMap(logger, ctx, f, selfId)
	if err == nil {
		t.Error("expected a hard failure for infrastructure error, but got nil")
	}
}
