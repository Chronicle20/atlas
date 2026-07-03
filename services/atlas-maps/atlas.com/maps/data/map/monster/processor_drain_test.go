package monster_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"atlas-maps/data/map/monster"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
)

// spawnPointDoc renders a JSON:API document for monster spawn points
// [from, to] on a single map. meta describes the current page/total so
// requests.DrainProvider can decide whether to keep paging.
func spawnPointDoc(from, to int, total, number, size, last int) string {
	var b strings.Builder
	for id := from; id <= to; id++ {
		if b.Len() > 0 {
			b.WriteString(",")
		}
		b.WriteString(fmt.Sprintf(
			`{"id":"%d","type":"monsters","attributes":{"template":100100,"mob_time":0,"team":-1,"cy":0,"f":0,"fh":1,"rx0":0,"rx1":0,"x":0,"y":0,"hide":false}}`,
			id,
		))
	}
	return fmt.Sprintf(
		`{"data":[%s],"meta":{"total":%d,"page":{"number":%d,"size":%d,"last":%d}}}`,
		b.String(), total, number, size, last,
	)
}

// TestSpawnPointProviderDrainsBeyondOnePage proves SpawnPointProvider (via
// requests.DrainProvider) fetches every page of a map's monster spawn
// points rather than stopping after the first response. atlas-data's GET
// /data/maps/{id}/monsters is now paginated (task-117); busy grinding/PQ
// maps commonly exceed the default page size. The fixture server serves 120
// spawn points across two pages of 100 -- only a genuine drain picks up
// spawn point id 120, which lives on page 2.
func TestSpawnPointProviderDrainsBeyondOnePage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		number, _ := strconv.Atoi(r.URL.Query().Get("page[number]"))
		w.Header().Set("Content-Type", "application/vnd.api+json")
		if number == 2 {
			_, _ = w.Write([]byte(spawnPointDoc(101, 120, 120, 2, 100, 2)))
			return
		}
		_, _ = w.Write([]byte(spawnPointDoc(1, 100, 120, 1, 100, 2)))
	}))
	defer srv.Close()
	t.Setenv("DATA_SERVICE_URL", srv.URL+"/")

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), ten)
	l, _ := test.NewNullLogger()

	sps, err := monster.NewProcessor(l, ctx).GetSpawnPoints(1)
	if err != nil {
		t.Fatal(err)
	}
	if len(sps) != 120 {
		t.Fatalf("expected 120 spawn points (full drain), got %d; a single-fetch implementation would return 100", len(sps))
	}

	foundLast := false
	for _, sp := range sps {
		if sp.Id == 120 {
			foundLast = true
		}
	}
	if !foundLast {
		t.Fatal("spawn point id 120 (page 2) must be present; single-fetch impl would miss it")
	}
}
