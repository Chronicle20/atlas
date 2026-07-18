package drop_test

import (
	"atlas-monsters/monster/drop"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// dropDoc renders a JSON:API document for monster drops [from, to]. meta
// describes the current page/total so requests.DrainProvider can decide
// whether to keep paging.
func dropDoc(from, to int, total, number, size, last int) string {
	var b strings.Builder
	for id := from; id <= to; id++ {
		if b.Len() > 0 {
			b.WriteString(",")
		}
		b.WriteString(fmt.Sprintf(
			`{"id":"%d","type":"drops","attributes":{"itemId":%d,"minimumQuantity":1,"maximumQuantity":1,"questId":0,"chance":50000}}`,
			id, 2000000+id,
		))
	}
	return fmt.Sprintf(
		`{"data":[%s],"meta":{"total":%d,"page":{"number":%d,"size":%d,"last":%d}}}`,
		b.String(), total, number, size, last,
	)
}

// TestGetByMonsterIdDrainsBeyondOnePage proves GetByMonsterId (via
// requests.DrainProvider) fetches every page of a monster's drop table
// rather than stopping after the first response. atlas-drop-information's
// GET /monsters/{monsterId}/drops is now paginated (task-117); this is a
// consumer-gate regression -- a single-fetch implementation would silently
// truncate at the server's default page size. The fixture server serves 260
// drops across two pages of 250 -- only a genuine drain picks up drop id
// 260, which lives on page 2.
func TestGetByMonsterIdDrainsBeyondOnePage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		number, _ := strconv.Atoi(r.URL.Query().Get("page[number]"))
		w.Header().Set("Content-Type", "application/vnd.api+json")
		if number == 2 {
			_, _ = w.Write([]byte(dropDoc(251, 260, 260, 2, 250, 2)))
			return
		}
		_, _ = w.Write([]byte(dropDoc(1, 250, 260, 1, 250, 2)))
	}))
	defer srv.Close()
	t.Setenv("DROPS_INFORMATION_SERVICE_URL", srv.URL+"/")

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), ten)
	l, _ := test.NewNullLogger()

	ds, err := drop.NewProcessor(l, ctx).GetByMonsterId(100100)
	if err != nil {
		t.Fatal(err)
	}
	if len(ds) != 260 {
		t.Fatalf("expected 260 drops (full drain), got %d; a single-fetch implementation would return 250", len(ds))
	}

	foundLast := false
	for _, d := range ds {
		if d.ItemId() == 2000260 {
			foundLast = true
		}
	}
	if !foundLast {
		t.Fatal("drop with itemId 2000260 (page 2) must be present; single-fetch impl would miss it")
	}
}
