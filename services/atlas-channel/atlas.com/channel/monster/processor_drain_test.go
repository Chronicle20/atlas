package monster_test

import (
	"atlas-channel/monster"
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
		b.WriteString(fmt.Sprintf(
			`{"id":"%d","type":"monsters","attributes":{"worldId":1,"channelId":1,"mapId":100000000,"instance":"00000000-0000-0000-0000-000000000000","monsterId":9300018,"controlCharacterId":0,"controllerHasAggro":false,"x":0,"y":0,"fh":0,"stance":0,"team":0,"maxHp":100,"hp":100,"maxMp":100,"mp":100,"damageEntries":[],"statusEffects":[]}}`,
			id,
		))
	}
	return fmt.Sprintf(
		`{"data":[%s],"meta":{"total":%d,"page":{"number":%d,"size":%d,"last":%d}}}`,
		b.String(), total, number, size, last,
	)
}

func monsterDrainTestServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		number, _ := strconv.Atoi(r.URL.Query().Get("page[number]"))
		w.Header().Set("Content-Type", "application/vnd.api+json")
		if number == 2 {
			_, _ = w.Write([]byte(monstersDoc(251, 300, 300, 2, 250, 2)))
			return
		}
		_, _ = w.Write([]byte(monstersDoc(1, 250, 300, 1, 250, 2)))
	}))
}

// TestInMapModelProviderDrainsBeyondOnePage proves InMapModelProvider (via
// requests.DrainProvider) fetches every page of monsters-in-a-map-instance
// rather than stopping after the first response. atlas-monsters' GET
// .../instances/{id}/monsters is now paginated (task-117); this is a hot
// path (spawn/movement/state on every channel spawn broadcast and AI tick)
// -- the fixture server serves 300 monsters across two pages of 250, so
// only a genuine drain picks up monster id 300, which lives on page 2.
func TestInMapModelProviderDrainsBeyondOnePage(t *testing.T) {
	srv := monsterDrainTestServer()
	defer srv.Close()
	t.Setenv("MONSTERS_SERVICE_URL", srv.URL+"/")

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), ten)
	l, _ := test.NewNullLogger()

	f := field.NewBuilder(world.Id(1), channel.Id(1), mapconst.Id(100000000)).SetInstance(uuid.Nil).Build()
	ms, err := monster.NewProcessor(l, ctx).InMapModelProvider(f)()
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 300 {
		t.Fatalf("expected 300 monsters (full drain), got %d; a single-fetch implementation would return 250", len(ms))
	}

	foundLast := false
	for _, m := range ms {
		if m.UniqueId() == 300 {
			foundLast = true
		}
	}
	if !foundLast {
		t.Fatal("monster unique id 300 (page 2) must be present; single-fetch impl would miss it")
	}
}

// TestInMapRectModelProviderDrainsBeyondOnePage is the same proof for the
// sibling in-rect arm. atlas-monsters preserves ascending-distance-from-
// center order across pages, so draining is safe even though the order is
// meaningful (not re-sorted by id).
func TestInMapRectModelProviderDrainsBeyondOnePage(t *testing.T) {
	srv := monsterDrainTestServer()
	defer srv.Close()
	t.Setenv("MONSTERS_SERVICE_URL", srv.URL+"/")

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), ten)
	l, _ := test.NewNullLogger()

	f := field.NewBuilder(world.Id(1), channel.Id(1), mapconst.Id(100000000)).SetInstance(uuid.Nil).Build()
	ms, err := monster.NewProcessor(l, ctx).InMapRectModelProvider(f, 0, 0, 1000, 1000, 0)()
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 300 {
		t.Fatalf("expected 300 monsters (full drain), got %d; a single-fetch implementation would return 250", len(ms))
	}
}
