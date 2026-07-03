package buffs_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"atlas-effective-stats/external/buffs"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
)

// buffDoc renders a JSON:API document for buffs with SourceId [from, to]
// belonging to a character. meta describes the current page/total so
// requests.DrainProvider can decide whether to keep paging.
func buffDoc(from, to int, total, number, size, last int) string {
	var b strings.Builder
	for id := from; id <= to; id++ {
		if b.Len() > 0 {
			b.WriteString(",")
		}
		b.WriteString(fmt.Sprintf(
			`{"id":"%d","type":"buffs","attributes":{"sourceId":%d,"duration":60000,"changes":[],"createdAt":"2024-01-01T00:00:00Z","expiresAt":"2024-01-01T01:00:00Z"}}`,
			id, id,
		))
	}
	return fmt.Sprintf(
		`{"data":[%s],"meta":{"total":%d,"page":{"number":%d,"size":%d,"last":%d}}}`,
		b.String(), total, number, size, last,
	)
}

// TestRequestCharacterBuffsDrainsBeyondOnePage proves RequestCharacterBuffs
// (via requests.DrainProvider) fetches every page of a character's buffs
// rather than stopping after the first response. atlas-buffs' GET
// /characters/{characterId}/buffs is now paginated (task-117); the fixture
// server serves 300 buffs across two pages of 250 -- only a genuine drain
// picks up buff sourceId 300, which lives on page 2. fetchBuffBonuses (the
// sole caller) needs every buff to compute stat bonuses correctly.
func TestRequestCharacterBuffsDrainsBeyondOnePage(t *testing.T) {
	characterId := uint32(42)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		number, _ := strconv.Atoi(r.URL.Query().Get("page[number]"))
		w.Header().Set("Content-Type", "application/vnd.api+json")
		if number == 2 {
			_, _ = w.Write([]byte(buffDoc(251, 300, 300, 2, 250, 2)))
			return
		}
		_, _ = w.Write([]byte(buffDoc(1, 250, 300, 1, 250, 2)))
	}))
	defer srv.Close()
	t.Setenv("BUFFS_SERVICE_URL", srv.URL+"/")

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), ten)
	l, _ := test.NewNullLogger()

	ms, err := buffs.RequestCharacterBuffs(characterId)(l, ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 300 {
		t.Fatalf("expected 300 buffs (full drain), got %d; a single-fetch implementation would return 250", len(ms))
	}

	foundLast := false
	for _, m := range ms {
		if m.SourceId == 300 {
			foundLast = true
		}
	}
	if !foundLast {
		t.Fatal("buff sourceId 300 (page 2) must be present; single-fetch impl would miss it")
	}
}
