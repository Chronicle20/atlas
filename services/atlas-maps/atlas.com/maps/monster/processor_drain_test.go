package monster_test

import (
	"atlas-maps/monster"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	mapconst "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// monstersDoc renders a JSON:API "monsters" document for unique ids
// [from, to]. meta describes the current page/total so
// requests.DrainProvider can decide whether to keep paging.
func monstersDoc(from, to int, total, number, size, last int) string {
	var b strings.Builder
	for id := from; id <= to; id++ {
		if b.Len() > 0 {
			b.WriteString(",")
		}
		b.WriteString(fmt.Sprintf(`{"id":"%d","type":"monsters"}`, id))
	}
	return fmt.Sprintf(
		`{"data":[%s],"meta":{"total":%d,"page":{"number":%d,"size":%d,"last":%d}}}`,
		b.String(), total, number, size, last,
	)
}

// TestCountInMapDrainsBeyondOnePage proves CountInMap (via
// requests.DrainProvider) counts every page of monsters-in-a-map-instance
// rather than stopping after the first response. atlas-monsters' GET
// .../instances/{id}/monsters is now paginated (task-117) -- the fixture
// server serves 300 monsters across two pages of 250, so only a genuine
// drain reports the true count of 300 (a single-fetch implementation would
// silently under-report 250).
func TestCountInMapDrainsBeyondOnePage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		number, _ := strconv.Atoi(r.URL.Query().Get("page[number]"))
		w.Header().Set("Content-Type", "application/vnd.api+json")
		if number == 2 {
			_, _ = w.Write([]byte(monstersDoc(251, 300, 300, 2, 250, 2)))
			return
		}
		_, _ = w.Write([]byte(monstersDoc(1, 250, 300, 1, 250, 2)))
	}))
	defer srv.Close()
	t.Setenv("MONSTERS_SERVICE_URL", srv.URL+"/")

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), ten)
	l, _ := test.NewNullLogger()

	f := field.NewBuilder(world.Id(1), channel.Id(1), mapconst.Id(100000000)).SetInstance(uuid.Nil).Build()
	count, err := monster.NewProcessor(l, ctx).CountInMap(uuid.New(), f)
	if err != nil {
		t.Fatal(err)
	}
	if count != 300 {
		t.Fatalf("expected count 300 (full drain), got %d; a single-fetch implementation would return 250", count)
	}
}
