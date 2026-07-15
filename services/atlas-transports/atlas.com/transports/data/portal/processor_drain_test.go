package portal_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"atlas-transports/data/portal"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
)

// portalDoc renders a JSON:API document for portals [from, to] on a single
// map. meta describes the current page/total so requests.DrainProvider can
// decide whether to keep paging.
func portalDoc(from, to int, total, number, size, last int) string {
	var b strings.Builder
	for id := from; id <= to; id++ {
		if b.Len() > 0 {
			b.WriteString(",")
		}
		b.WriteString(fmt.Sprintf(
			`{"id":"%d","type":"portals","attributes":{"name":"p%d","target":"","type":0,"x":0,"y":0,"targetMapId":999999999,"scriptName":""}}`,
			id, id,
		))
	}
	return fmt.Sprintf(
		`{"data":[%s],"meta":{"total":%d,"page":{"number":%d,"size":%d,"last":%d}}}`,
		b.String(), total, number, size, last,
	)
}

// TestInMapProviderDrainsBeyondOnePage proves InMapProvider (via
// requests.DrainProvider) fetches every page of a map's portals rather than
// stopping after the first response. atlas-data's GET
// /data/maps/{id}/portals is now paginated (task-117); the fixture server
// serves 60 portals across two pages of 50 -- only a genuine drain picks up
// portal id 60, which lives on page 2.
func TestInMapProviderDrainsBeyondOnePage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		number, _ := strconv.Atoi(r.URL.Query().Get("page[number]"))
		w.Header().Set("Content-Type", "application/vnd.api+json")
		if number == 2 {
			_, _ = w.Write([]byte(portalDoc(51, 60, 60, 2, 50, 2)))
			return
		}
		_, _ = w.Write([]byte(portalDoc(1, 50, 60, 1, 50, 2)))
	}))
	defer srv.Close()
	t.Setenv("DATA_SERVICE_URL", srv.URL+"/")

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), ten)
	l, _ := test.NewNullLogger()

	ms, err := portal.NewProcessor(l, ctx).InMapProvider(_map.Id(1))()
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 60 {
		t.Fatalf("expected 60 portals (full drain), got %d; a single-fetch implementation would return 50", len(ms))
	}

	foundLast := false
	for _, m := range ms {
		if m.Id() == 60 {
			foundLast = true
		}
	}
	if !foundLast {
		t.Fatal("portal id 60 (page 2) must be present; single-fetch impl would miss it")
	}
}
