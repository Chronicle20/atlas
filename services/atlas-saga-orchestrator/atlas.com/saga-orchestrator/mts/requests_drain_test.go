package mts_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"atlas-saga-orchestrator/mts"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
)

// holdingsDoc renders a JSON:API "holdings" list response for holdings
// [from, to]. Each holding's id is deterministic ("00000000-...-<n>") so the
// test can assert a page-2-only holding is found by id.
func holdingsDoc(from, to int, total, number, size, last int) string {
	var b strings.Builder
	for i := from; i <= to; i++ {
		if b.Len() > 0 {
			b.WriteString(",")
		}
		id := fmt.Sprintf("00000000-0000-0000-0000-%012d", i)
		b.WriteString(fmt.Sprintf(
			`{"id":"%s","type":"holdings","attributes":{"worldId":1,"ownerId":100100,"origin":"purchased","templateId":1302000,"quantity":1}}`,
			id,
		))
	}
	return fmt.Sprintf(
		`{"data":[%s],"meta":{"total":%d,"page":{"number":%d,"size":%d,"last":%d}}}`,
		b.String(), total, number, size, last,
	)
}

// TestRequestHoldingsDrainsBeyondOnePage proves RequestHoldings (via
// requests.DrainProvider) fetches every page of a character's holdings rather
// than stopping after the first response. atlas-mts's GET
// /characters/{characterId}/mts/holding is now paginated (task-117);
// expandWithdrawFromMts linear-searches the result by HoldingId, a genuine
// semantic-all consumer. The fixture server serves 260 holdings across two
// pages of 250 -- only a genuine drain can find a holding whose id lives on
// page 2.
func TestRequestHoldingsDrainsBeyondOnePage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		number, _ := strconv.Atoi(r.URL.Query().Get("page[number]"))
		w.Header().Set("Content-Type", "application/vnd.api+json")
		if number == 2 {
			_, _ = w.Write([]byte(holdingsDoc(251, 260, 260, 2, 250, 2)))
			return
		}
		_, _ = w.Write([]byte(holdingsDoc(1, 250, 260, 1, 250, 2)))
	}))
	defer srv.Close()
	t.Setenv("MTS_SERVICE_URL", srv.URL+"/")

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), ten)
	l, _ := test.NewNullLogger()

	holdings, err := mts.RequestHoldings(l, ctx)(100100, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(holdings) != 260 {
		t.Fatalf("expected 260 holdings (full drain), got %d; a single-fetch implementation would return 250", len(holdings))
	}
	wantId := "00000000-0000-0000-0000-000000000260"
	found := false
	for _, h := range holdings {
		if h.Id == wantId {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected holding %q (page 2) to be found via drain", wantId)
	}
}
