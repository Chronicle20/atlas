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

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// rectDoc renders a JSON:API "monsters" document carrying position
// attributes, so the rect result's X/Y round-trip can be asserted.
// monstersDoc (processor_drain_test.go) does not carry attributes, so it
// cannot be reused for this test's X/Y assertions.
func rectDoc(from, to int, total, number, size, last int) string {
	body := ""
	for id := from; id <= to; id++ {
		if body != "" {
			body += ","
		}
		body += fmt.Sprintf(`{"id":"%d","type":"monsters","attributes":{"x":%d,"y":300}}`, id, 400+id)
	}
	return fmt.Sprintf(
		`{"data":[%s],"meta":{"total":%d,"page":{"number":%d,"size":%d,"last":%d}}}`,
		body, total, number, size, last,
	)
}

// TestGetInMapRect_DrainsAllPagesAndCarriesBounds asserts the rect query is
// drained across pages (a truncated page-1 result would silently under-apply a
// mist) and that the request carries the inclusive bounds and the limit.
func TestGetInMapRect_DrainsAllPagesAndCarriesBounds(t *testing.T) {
	var firstQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if firstQuery == "" {
			firstQuery = r.URL.RawQuery
		}
		number, _ := strconv.Atoi(r.URL.Query().Get("page[number]"))
		w.Header().Set("Content-Type", "application/vnd.api+json")
		if number == 2 {
			_, _ = w.Write([]byte(rectDoc(251, 300, 300, 2, 250, 2)))
			return
		}
		_, _ = w.Write([]byte(rectDoc(1, 250, 300, 1, 250, 2)))
	}))
	defer srv.Close()
	t.Setenv("MONSTERS_SERVICE_URL", srv.URL+"/")

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), ten)
	l, _ := test.NewNullLogger()
	f := field.NewBuilder(0, 0, 100000000).SetInstance(uuid.Nil).Build()

	got, err := monster.NewProcessor(l, ctx).GetInMapRect(f, 390, 218, 610, 383, 0)
	if err != nil {
		t.Fatalf("GetInMapRect: %v", err)
	}
	if len(got) != 300 {
		t.Fatalf("len(got) = %d, want 300 (drain must not stop at page 1)", len(got))
	}
	// `max=0`, not `limit=0` -- a `limit` param is rejected with 400 by the
	// server's paginate.ParseParams (task-117 ban), which is exactly how this
	// call silently failed in production. See requests_rect_url_test.go.
	for _, want := range []string{"x1=390", "y1=218", "x2=610", "y2=383", "max=0"} {
		if !strings.Contains(firstQuery, want) {
			t.Fatalf("query %q missing %q", firstQuery, want)
		}
	}
	if got[0].Id != "1" {
		t.Fatalf("got[0].Id = %q, want \"1\" (the monster unique id)", got[0].Id)
	}
	if got[0].X != 401 || got[0].Y != 300 {
		t.Fatalf("got[0] position = (%d,%d), want (401,300)", got[0].X, got[0].Y)
	}
}
