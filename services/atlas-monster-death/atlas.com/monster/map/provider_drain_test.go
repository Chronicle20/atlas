package _map_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	_map "atlas-monster-death/map"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	mapconst "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// charactersDoc renders a JSON:API "characters" document for character ids
// [from, to]. meta describes the current page/total so
// requests.DrainProvider can decide whether to keep paging.
func charactersDoc(from, to int, total, number, size, last int) string {
	var b strings.Builder
	for id := from; id <= to; id++ {
		if b.Len() > 0 {
			b.WriteString(",")
		}
		b.WriteString(fmt.Sprintf(`{"id":"%d","type":"characters"}`, id))
	}
	return fmt.Sprintf(
		`{"data":[%s],"meta":{"total":%d,"page":{"number":%d,"size":%d,"last":%d}}}`,
		b.String(), total, number, size, last,
	)
}

// TestCharacterIdsInFieldModelProviderDrainsBeyondOnePage proves
// CharacterIdsInFieldModelProvider (via requests.DrainProvider) fetches
// every page of characters-in-a-map-instance rather than stopping after the
// first response. atlas-maps' GET
// /worlds/{w}/channels/{c}/maps/{m}/instances/{i}/characters is now
// paginated (task-117); this consumer needs the complete set to distribute
// mob-kill drops/quest credit to everyone present -- the fixture server
// serves 300 characters across two pages of 250, so only a genuine drain
// picks up character id 300, which lives on page 2.
func TestCharacterIdsInFieldModelProviderDrainsBeyondOnePage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		number, _ := strconv.Atoi(r.URL.Query().Get("page[number]"))
		w.Header().Set("Content-Type", "application/vnd.api+json")
		if number == 2 {
			_, _ = w.Write([]byte(charactersDoc(251, 300, 300, 2, 250, 2)))
			return
		}
		_, _ = w.Write([]byte(charactersDoc(1, 250, 300, 1, 250, 2)))
	}))
	defer srv.Close()
	t.Setenv("MAPS_SERVICE_URL", srv.URL+"/")

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), ten)
	l, _ := test.NewNullLogger()

	f := field.NewBuilder(world.Id(1), channel.Id(1), mapconst.Id(100000000)).SetInstance(uuid.Nil).Build()
	ids, err := _map.CharacterIdsInFieldModelProvider(l)(ctx)(f)()
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 300 {
		t.Fatalf("expected 300 characters (full drain), got %d; a single-fetch implementation would return 250", len(ids))
	}

	foundLast := false
	for _, id := range ids {
		if id == 300 {
			foundLast = true
		}
	}
	if !foundLast {
		t.Fatal("character id 300 (page 2) must be present; single-fetch impl would miss it")
	}
}
