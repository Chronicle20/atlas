package summon_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"atlas-channel/summon"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	mapconst "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
)

// summonsDoc renders a JSON:API "summons" document for summon ids
// [from, to]. meta describes the current page/total so
// requests.DrainProvider can decide whether to keep paging.
func summonsDoc(from, to int, total, number, size, last int) string {
	var b strings.Builder
	for id := from; id <= to; id++ {
		if b.Len() > 0 {
			b.WriteString(",")
		}
		b.WriteString(fmt.Sprintf(
			`{"id":"%d","type":"summons","attributes":{"ownerCharacterId":42,"skillId":3111002,"skillLevel":20,"summonType":"PUPPET","movementType":0,"x":100,"y":-50}}`,
			id,
		))
	}
	return fmt.Sprintf(
		`{"data":[%s],"meta":{"total":%d,"page":{"number":%d,"size":%d,"last":%d}}}`,
		b.String(), total, number, size, last,
	)
}

// TestInMapModelProviderDrainsBeyondOnePage proves InMapModelProvider (via
// requests.DrainProvider) fetches every page of summons-in-a-map-instance
// rather than stopping after the first response. atlas-summons' GET
// .../instances/{id}/summons is now paginated (task-117); this consumer
// replays every existing summon to a character entering the map -- the
// fixture server serves 300 summons across two pages of 250, so only a
// genuine drain picks up summon id 300, which lives on page 2.
func TestInMapModelProviderDrainsBeyondOnePage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		number, _ := strconv.Atoi(r.URL.Query().Get("page[number]"))
		w.Header().Set("Content-Type", "application/vnd.api+json")
		if number == 2 {
			_, _ = w.Write([]byte(summonsDoc(251, 300, 300, 2, 250, 2)))
			return
		}
		_, _ = w.Write([]byte(summonsDoc(1, 250, 300, 1, 250, 2)))
	}))
	defer srv.Close()
	t.Setenv("SUMMONS_SERVICE_URL", srv.URL+"/")

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), ten)
	l, _ := test.NewNullLogger()

	f := field.NewBuilder(world.Id(1), channel.Id(1), mapconst.Id(100000000)).SetInstance(uuid.Nil).Build()
	ms, err := summon.NewProcessor(l, ctx).InMapModelProvider(f)()
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 300 {
		t.Fatalf("expected 300 summons (full drain), got %d; a single-fetch implementation would return 250", len(ms))
	}

	foundLast := false
	for _, m := range ms {
		if m.Id() == 300 {
			foundLast = true
		}
	}
	if !foundLast {
		t.Fatal("summon id 300 (page 2) must be present; single-fetch impl would miss it")
	}
}
