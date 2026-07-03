package key_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"atlas-channel/character/key"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
)

// keyDoc renders a JSON:API document for key bindings [from, to] belonging
// to a character. meta describes the current page/total so
// requests.DrainProvider can decide whether to keep paging.
func keyDoc(from, to int, total, number, size, last int) string {
	var b strings.Builder
	for id := from; id <= to; id++ {
		if b.Len() > 0 {
			b.WriteString(",")
		}
		b.WriteString(fmt.Sprintf(
			`{"id":"%d","type":"keys","attributes":{"type":4,"action":0}}`,
			id,
		))
	}
	return fmt.Sprintf(
		`{"data":[%s],"meta":{"total":%d,"page":{"number":%d,"size":%d,"last":%d}}}`,
		b.String(), total, number, size, last,
	)
}

// TestByCharacterIdProviderDrainsBeyondOnePage proves ByCharacterIdProvider
// (via requests.DrainProvider) fetches every page of a character's key
// bindings rather than stopping after the first response. atlas-keys' GET
// /characters/{characterId}/keys is now paginated (task-117); the fixture
// server serves 300 key bindings across two pages of 250 -- only a genuine
// drain picks up key id 300, which lives on page 2.
func TestByCharacterIdProviderDrainsBeyondOnePage(t *testing.T) {
	characterId := uint32(42)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		number, _ := strconv.Atoi(r.URL.Query().Get("page[number]"))
		w.Header().Set("Content-Type", "application/vnd.api+json")
		if number == 2 {
			_, _ = w.Write([]byte(keyDoc(251, 300, 300, 2, 250, 2)))
			return
		}
		_, _ = w.Write([]byte(keyDoc(1, 250, 300, 1, 250, 2)))
	}))
	defer srv.Close()
	t.Setenv("KEYS_SERVICE_URL", srv.URL+"/")

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), ten)
	l, _ := test.NewNullLogger()

	ms, err := key.NewProcessor(l, ctx).ByCharacterIdProvider(characterId)()
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 300 {
		t.Fatalf("expected 300 key bindings (full drain), got %d; a single-fetch implementation would return 250", len(ms))
	}

	foundLast := false
	for _, m := range ms {
		if m.Key() == 300 {
			foundLast = true
		}
	}
	if !foundLast {
		t.Fatal("key id 300 (page 2) must be present; single-fetch impl would miss it")
	}
}
