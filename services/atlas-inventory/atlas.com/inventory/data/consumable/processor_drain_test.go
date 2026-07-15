package consumable_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"atlas-inventory/data/consumable"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
)

// consumableDoc renders a JSON:API document for rechargeable consumables
// [from, to]. meta describes the current page/total so
// requests.DrainProvider can decide whether to keep paging.
func consumableDoc(from, to int, total, number, size, last int) string {
	var b strings.Builder
	for id := from; id <= to; id++ {
		if b.Len() > 0 {
			b.WriteString(",")
		}
		b.WriteString(fmt.Sprintf(
			`{"id":"%d","type":"consumables","attributes":{"price":1,"slotMax":100}}`,
			id,
		))
	}
	return fmt.Sprintf(
		`{"data":[%s],"meta":{"total":%d,"page":{"number":%d,"size":%d,"last":%d}}}`,
		b.String(), total, number, size, last,
	)
}

// TestGetRechargeableDrainsBeyondOnePage proves GetRechargeable (via
// requests.DrainProvider) fetches every page of the rechargeable-consumable
// filter set rather than stopping after the first response. atlas-data's
// GET /data/consumables?filter[rechargeable]=true is now paginated
// (task-117); the fixture server serves 60 items across two pages of 50 --
// only a genuine drain picks up item id 60, which lives on page 2.
func TestGetRechargeableDrainsBeyondOnePage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		number, _ := strconv.Atoi(r.URL.Query().Get("page[number]"))
		w.Header().Set("Content-Type", "application/vnd.api+json")
		if number == 2 {
			_, _ = w.Write([]byte(consumableDoc(51, 60, 60, 2, 50, 2)))
			return
		}
		_, _ = w.Write([]byte(consumableDoc(1, 50, 60, 1, 50, 2)))
	}))
	defer srv.Close()
	t.Setenv("DATA_SERVICE_URL", srv.URL+"/")

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), ten)
	l, _ := test.NewNullLogger()

	ms, err := consumable.NewProcessor(l, ctx).GetRechargeable()
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 60 {
		t.Fatalf("expected 60 rechargeable consumables (full drain), got %d; a single-fetch implementation would return 50", len(ms))
	}

	foundLast := false
	for _, m := range ms {
		if m.Id() == 60 {
			foundLast = true
		}
	}
	if !foundLast {
		t.Fatal("consumable id 60 (page 2) must be present; single-fetch impl would miss it")
	}
}
