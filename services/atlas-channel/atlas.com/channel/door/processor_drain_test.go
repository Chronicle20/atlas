package door_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"atlas-channel/door"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	mapconst "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
)

// doorsDoc renders a JSON:API "doors" document for pair ids [from, to].
// meta describes the current page/total so requests.DrainProvider can
// decide whether to keep paging.
func doorsDoc(from, to int, total, number, size, last int) string {
	var b strings.Builder
	for id := from; id <= to; id++ {
		if b.Len() > 0 {
			b.WriteString(",")
		}
		b.WriteString(fmt.Sprintf(
			`{"id":"%d","type":"doors","attributes":{"areaDoorId":%d,"townDoorId":%d,"pairId":%d,"ownerCharacterId":42,"partyId":0,"worldId":1,"channelId":1,"mapId":100000000,"instance":"00000000-0000-0000-0000-000000000000","townMapId":104000000,"slot":0,"townPortalId":0,"areaX":0,"areaY":0,"townX":0,"townY":0,"skillId":1230003,"skillLevel":1,"expiresAt":"2024-01-01T00:00:00Z"}}`,
			id, id, id+1, id,
		))
	}
	return fmt.Sprintf(
		`{"data":[%s],"meta":{"total":%d,"page":{"number":%d,"size":%d,"last":%d}}}`,
		b.String(), total, number, size, last,
	)
}

func doorDrainTestServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		number, _ := strconv.Atoi(r.URL.Query().Get("page[number]"))
		w.Header().Set("Content-Type", "application/vnd.api+json")
		if number == 2 {
			_, _ = w.Write([]byte(doorsDoc(251, 300, 300, 2, 250, 2)))
			return
		}
		_, _ = w.Write([]byte(doorsDoc(1, 250, 300, 1, 250, 2)))
	}))
}

// TestInFieldModelProviderDrainsBeyondOnePage proves InFieldModelProvider
// (via requests.DrainProvider) fetches every page of doors-in-a-map-
// instance rather than stopping after the first response. atlas-doors' GET
// .../instances/{id}/doors is now paginated (task-117); this is a hot path
// (door spawn/state on channel spawn broadcast, ForEachInMap,
// GetByOwnerOnMap) -- the fixture server serves 300 doors across two pages
// of 250, so only a genuine drain picks up pair id 300, which lives on
// page 2.
func TestInFieldModelProviderDrainsBeyondOnePage(t *testing.T) {
	srv := doorDrainTestServer()
	defer srv.Close()
	t.Setenv("DOORS_SERVICE_URL", srv.URL+"/")

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), ten)
	l, _ := test.NewNullLogger()

	f := field.NewBuilder(world.Id(1), channel.Id(1), mapconst.Id(100000000)).SetInstance(uuid.Nil).Build()
	ms, err := door.NewProcessor(l, ctx).InFieldModelProvider(f)()
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 300 {
		t.Fatalf("expected 300 doors (full drain), got %d; a single-fetch implementation would return 250", len(ms))
	}

	foundLast := false
	for _, m := range ms {
		if m.PairId() == 300 {
			foundLast = true
		}
	}
	if !foundLast {
		t.Fatal("door pair id 300 (page 2) must be present; single-fetch impl would miss it")
	}
}

// TestByOwnerModelProviderDrainsBeyondOnePage is the same proof for the
// sibling by-owner arm.
func TestByOwnerModelProviderDrainsBeyondOnePage(t *testing.T) {
	srv := doorDrainTestServer()
	defer srv.Close()
	t.Setenv("DOORS_SERVICE_URL", srv.URL+"/")

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), ten)
	l, _ := test.NewNullLogger()

	ms, err := door.NewProcessor(l, ctx).ByOwnerModelProvider(42)()
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 300 {
		t.Fatalf("expected 300 doors (full drain), got %d; a single-fetch implementation would return 250", len(ms))
	}
}
