package wish_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"atlas-channel/mts/wish"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
)

// wishListDoc renders a JSON:API "wish-entries" list response for entries
// [from, to] belonging to a character. meta describes the current page/total
// so requests.DrainProvider can decide whether to keep paging.
func wishListDoc(from, to int, total, number, size, last int) string {
	var b strings.Builder
	for i := from; i <= to; i++ {
		if b.Len() > 0 {
			b.WriteString(",")
		}
		b.WriteString(fmt.Sprintf(
			`{"id":"%s","type":"wish-entries","attributes":{"worldId":1,"serial":%d,"characterId":9001,"itemId":1302000,"listingSerial":0,"price":1500,"count":1}}`,
			uuid.New().String(), 5000000+i,
		))
	}
	return fmt.Sprintf(
		`{"data":[%s],"meta":{"total":%d,"page":{"number":%d,"size":%d,"last":%d}}}`,
		b.String(), total, number, size, last,
	)
}

func twoPageDrainServer(doc func(from, to, total, number, size, last int) string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		number, _ := strconv.Atoi(r.URL.Query().Get("page[number]"))
		w.Header().Set("Content-Type", "application/vnd.api+json")
		if number == 2 {
			_, _ = w.Write([]byte(doc(251, 300, 300, 2, 250, 2)))
			return
		}
		_, _ = w.Write([]byte(doc(1, 250, 300, 1, 250, 2)))
	}))
}

func newTestContext(t *testing.T) context.Context {
	t.Helper()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	return tenant.WithContext(context.Background(), ten)
}

// TestGetByCharacterProviderDrainsBeyondOnePage proves GetByCharacterProvider
// (via requests.DrainProvider) fetches every page of a character's wishlist
// rather than stopping after the first response. atlas-mts's
// GET /characters/{characterId}/mts/wishlist is now paginated (task-117); the
// fixture server serves 300 items across two pages of 250 -- only a genuine
// drain picks up the last item, which lives on page 2. GetByCharacterItem and
// GetByCharacterSerial linear-search this same result, so a truncated drain
// would silently fail to resolve entries living past item 250.
func TestGetByCharacterProviderDrainsBeyondOnePage(t *testing.T) {
	srv := twoPageDrainServer(wishListDoc)
	defer srv.Close()
	t.Setenv("MTS_SERVICE_URL", srv.URL+"/")

	ctx := newTestContext(t)
	l, _ := test.NewNullLogger()

	ms, err := wish.NewProcessor(l, ctx).GetByCharacter(9001)
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 300 {
		t.Fatalf("expected 300 wish entries (full drain), got %d; a single-fetch implementation would return 250", len(ms))
	}
}

// TestGetByCharacterAndTypeDrainsBeyondOnePage mirrors
// TestGetByCharacterProviderDrainsBeyondOnePage for the ?type= filtered branch
// (the Cart / Wanted MTS views), which shares the same paginated upstream list.
func TestGetByCharacterAndTypeDrainsBeyondOnePage(t *testing.T) {
	srv := twoPageDrainServer(wishListDoc)
	defer srv.Close()
	t.Setenv("MTS_SERVICE_URL", srv.URL+"/")

	ctx := newTestContext(t)
	l, _ := test.NewNullLogger()

	ms, err := wish.NewProcessor(l, ctx).GetByCharacterAndType(9001, wish.TypeCart)
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 300 {
		t.Fatalf("expected 300 wish entries (full drain), got %d; a single-fetch implementation would return 250", len(ms))
	}
}

// TestGetWantedByWorldDrainsBeyondOnePage mirrors
// TestGetByCharacterProviderDrainsBeyondOnePage for the cross-character
// world-wide Wanted browse tab (GET /worlds/{worldId}/mts/wishlist), which is
// paginated server-side independently of the per-character wishlist route.
func TestGetWantedByWorldDrainsBeyondOnePage(t *testing.T) {
	srv := twoPageDrainServer(wishListDoc)
	defer srv.Close()
	t.Setenv("MTS_SERVICE_URL", srv.URL+"/")

	ctx := newTestContext(t)
	l, _ := test.NewNullLogger()

	ms, err := wish.NewProcessor(l, ctx).GetWantedByWorld(1)
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 300 {
		t.Fatalf("expected 300 want-ads (full drain), got %d; a single-fetch implementation would return 250", len(ms))
	}
}
