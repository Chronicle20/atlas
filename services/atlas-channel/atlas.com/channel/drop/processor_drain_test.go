package drop_test

import (
	"atlas-channel/drop"
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

// dropsDoc renders a JSON:API "drops" document for drop ids [from, to].
// meta describes the current page/total so requests.DrainProvider can
// decide whether to keep paging.
func dropsDoc(from, to int, total, number, size, last int) string {
	var b strings.Builder
	for id := from; id <= to; id++ {
		if b.Len() > 0 {
			b.WriteString(",")
		}
		b.WriteString(fmt.Sprintf(
			`{"id":"%d","type":"drops","attributes":{"worldId":1,"channelId":1,"mapId":100000000,"instance":"00000000-0000-0000-0000-000000000000","itemId":1000000,"quantity":1,"meso":0,"type":0,"x":100,"y":200,"ownerId":0,"ownerPartyId":0,"dropTime":"2024-01-01T00:00:00Z","dropperId":0,"dropperX":0,"dropperY":0,"characterDrop":false,"mod":0}}`,
			id,
		))
	}
	return fmt.Sprintf(
		`{"data":[%s],"meta":{"total":%d,"page":{"number":%d,"size":%d,"last":%d}}}`,
		b.String(), total, number, size, last,
	)
}

// TestInMapModelProviderDrainsBeyondOnePage proves InMapModelProvider (via
// requests.DrainProvider) fetches every page of drops-in-a-map-instance
// rather than stopping after the first response. atlas-drops' GET
// .../instances/{id}/drops is now paginated (task-117); this is a hot path
// (drop spawn state on channel spawn broadcast, ForEachInMap reservation
// logic) -- the fixture server serves 300 drops across two pages of 250,
// so only a genuine drain picks up drop id 300, which lives on page 2.
func TestInMapModelProviderDrainsBeyondOnePage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		number, _ := strconv.Atoi(r.URL.Query().Get("page[number]"))
		w.Header().Set("Content-Type", "application/vnd.api+json")
		if number == 2 {
			_, _ = w.Write([]byte(dropsDoc(251, 300, 300, 2, 250, 2)))
			return
		}
		_, _ = w.Write([]byte(dropsDoc(1, 250, 300, 1, 250, 2)))
	}))
	defer srv.Close()
	t.Setenv("DROPS_SERVICE_URL", srv.URL+"/")

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), ten)
	l, _ := test.NewNullLogger()

	f := field.NewBuilder(world.Id(1), channel.Id(1), mapconst.Id(100000000)).SetInstance(uuid.Nil).Build()
	ds, err := drop.NewProcessor(l, ctx).InMapModelProvider(f)()
	if err != nil {
		t.Fatal(err)
	}
	if len(ds) != 300 {
		t.Fatalf("expected 300 drops (full drain), got %d; a single-fetch implementation would return 250", len(ds))
	}

	foundLast := false
	for _, d := range ds {
		if d.Id() == 300 {
			foundLast = true
		}
	}
	if !foundLast {
		t.Fatal("drop id 300 (page 2) must be present; single-fetch impl would miss it")
	}
}
