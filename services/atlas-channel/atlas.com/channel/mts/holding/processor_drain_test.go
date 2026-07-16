package holding_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"atlas-channel/mts/holding"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
)

// holdingsDoc renders a JSON:API "holdings" list response for holdings
// [from, to] belonging to a character. meta describes the current page/total
// so requests.DrainProvider can decide whether to keep paging.
func holdingsDoc(from, to int, total, number, size, last int) string {
	var b strings.Builder
	for i := from; i <= to; i++ {
		if b.Len() > 0 {
			b.WriteString(",")
		}
		b.WriteString(fmt.Sprintf(
			`{"id":"%s","type":"holdings","attributes":{"worldId":1,"itcSn":%d,"ownerId":100100,"origin":"purchased","templateId":1302000,"quantity":1}}`,
			uuid.New().String(), 5000000+i,
		))
	}
	return fmt.Sprintf(
		`{"data":[%s],"meta":{"total":%d,"page":{"number":%d,"size":%d,"last":%d}}}`,
		b.String(), total, number, size, last,
	)
}

// TestGetByCharacterProviderDrainsBeyondOnePage proves GetByCharacterProvider
// (via requests.DrainProvider) fetches every page of a character's take-home
// holdings rather than stopping after the first response. atlas-mts's
// GET /characters/{characterId}/mts/holding is now paginated (task-117); the
// fixture server serves 300 items across two pages of 250 -- only a genuine
// drain picks up the last item, which lives on page 2.
func TestGetByCharacterProviderDrainsBeyondOnePage(t *testing.T) {
	characterId := uint32(100100)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		number, _ := strconv.Atoi(r.URL.Query().Get("page[number]"))
		w.Header().Set("Content-Type", "application/vnd.api+json")
		if number == 2 {
			_, _ = w.Write([]byte(holdingsDoc(251, 300, 300, 2, 250, 2)))
			return
		}
		_, _ = w.Write([]byte(holdingsDoc(1, 250, 300, 1, 250, 2)))
	}))
	defer srv.Close()
	t.Setenv("MTS_SERVICE_URL", srv.URL+"/")

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), ten)
	l, _ := test.NewNullLogger()

	ms, err := holding.NewProcessor(l, ctx).GetByCharacter(characterId)
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 300 {
		t.Fatalf("expected 300 holdings (full drain), got %d; a single-fetch implementation would return 250", len(ms))
	}
}
