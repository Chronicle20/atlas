package buff_test

import (
	"atlas-query-aggregator/buff"
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
			`{"id":"%d","type":"buffs","attributes":{"sourceId":%d,"duration":60000,"createdAt":"2024-01-01T00:00:00Z","expiresAt":"2999-01-01T01:00:00Z"}}`,
			id, id,
		))
	}
	return fmt.Sprintf(
		`{"data":[%s],"meta":{"total":%d,"page":{"number":%d,"size":%d,"last":%d}}}`,
		b.String(), total, number, size, last,
	)
}

// TestGetBuffsByCharacterDrainsBeyondOnePage proves GetBuffsByCharacter
// (via requests.DrainProvider) fetches every page of a character's buffs
// rather than stopping after the first response. atlas-buffs' GET
// /characters/{characterId}/buffs is now paginated (task-117); the fixture
// server serves 300 buffs across two pages of 250 -- only a genuine drain
// picks up buff sourceId 300, which lives on page 2. HasActiveBuff (the
// primary caller) scans every buff for a matching source.
func TestGetBuffsByCharacterDrainsBeyondOnePage(t *testing.T) {
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

	p := buff.NewProcessor(l, ctx)

	ms, err := p.GetBuffsByCharacter(characterId)()
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 300 {
		t.Fatalf("expected 300 buffs (full drain), got %d; a single-fetch implementation would return 250", len(ms))
	}

	// HasActiveBuff must find sourceId 300 (page 2); a single-fetch
	// implementation stopping at page 1 (sourceId 1-250) would report false.
	has, err := p.HasActiveBuff(characterId, 300)()
	if err != nil {
		t.Fatal(err)
	}
	if !has {
		t.Fatal("HasActiveBuff(300) must be true; buff sourceId 300 lives on page 2 and a single-fetch impl would miss it")
	}
}
