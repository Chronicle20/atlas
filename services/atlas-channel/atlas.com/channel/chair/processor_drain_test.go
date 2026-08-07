package chair_test

import (
	"atlas-channel/chair"
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

// chairsDoc renders a JSON:API "chairs" document for character ids
// [from, to] (one occupied chair per character). meta describes the
// current page/total so requests.DrainProvider can decide whether to keep
// paging.
func chairsDoc(from, to int, total, number, size, last int) string {
	var b strings.Builder
	for id := from; id <= to; id++ {
		if b.Len() > 0 {
			b.WriteString(",")
		}
		b.WriteString(fmt.Sprintf(
			`{"id":"%d","type":"chairs","attributes":{"type":"ITEM","characterId":%d}}`,
			id, id,
		))
	}
	return fmt.Sprintf(
		`{"data":[%s],"meta":{"total":%d,"page":{"number":%d,"size":%d,"last":%d}}}`,
		b.String(), total, number, size, last,
	)
}

// TestInMapModelProviderDrainsBeyondOnePage proves InMapModelProvider (via
// requests.DrainProvider) fetches every page of occupied-chairs-in-a-map-
// instance rather than stopping after the first response, and that the
// consumer now hits the corrected /instances/{id}/chairs URL (the old
// format string, missing that segment entirely, would 404 against every
// real atlas-chairs deployment). The fixture server serves 300 chairs
// across two pages of 250, so only a genuine drain picks up character id
// 300, which lives on page 2.
func TestInMapModelProviderDrainsBeyondOnePage(t *testing.T) {
	var sawPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawPath = r.URL.Path
		number, _ := strconv.Atoi(r.URL.Query().Get("page[number]"))
		w.Header().Set("Content-Type", "application/vnd.api+json")
		if number == 2 {
			_, _ = w.Write([]byte(chairsDoc(251, 300, 300, 2, 250, 2)))
			return
		}
		_, _ = w.Write([]byte(chairsDoc(1, 250, 300, 1, 250, 2)))
	}))
	defer srv.Close()
	t.Setenv("CHAIRS_SERVICE_URL", srv.URL+"/")

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), ten)
	l, _ := test.NewNullLogger()

	f := field.NewBuilder(world.Id(1), channel.Id(1), mapconst.Id(100000000)).SetInstance(uuid.Nil).Build()
	ms, err := chair.NewProcessor(l, ctx).InMapModelProvider(f)()
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 300 {
		t.Fatalf("expected 300 chairs (full drain), got %d; a single-fetch implementation would return 250", len(ms))
	}

	foundLast := false
	for _, m := range ms {
		if m.CharacterId() == 300 {
			foundLast = true
		}
	}
	if !foundLast {
		t.Fatal("chair for character 300 (page 2) must be present; single-fetch impl would miss it")
	}

	wantPath := fmt.Sprintf("/worlds/1/channels/1/maps/100000000/instances/%s/chairs", uuid.Nil.String())
	if sawPath != wantPath {
		t.Fatalf("expected request path %q (with /instances/{id} segment), got %q", wantPath, sawPath)
	}
}
