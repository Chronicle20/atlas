package world_test

import (
	"atlas-login/world"
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

// worldDoc renders a JSON:API document for worlds [from, to]. Each world's
// name is unique ("World-<n>") so the test can assert presence of a
// page-2-only item without relying on world.Id (a byte, which would
// overflow/truncate for ids above 255 - the fixture intentionally uses ids
// that fit, but identifies items by name to keep the assertion robust).
func worldDoc(from, to int, total, number, size, last int) string {
	var b strings.Builder
	for id := from; id <= to; id++ {
		if b.Len() > 0 {
			b.WriteString(",")
		}
		b.WriteString(fmt.Sprintf(
			`{"id":"%d","type":"worlds","attributes":{"name":"World-%d","state":0,"message":"","eventMessage":"","recommended":false,"recommendedMessage":"","capacityStatus":0,"expRate":1,"mesoRate":1,"itemDropRate":1,"questExpRate":1}}`,
			id%256, id,
		))
	}
	return fmt.Sprintf(
		`{"data":[%s],"meta":{"total":%d,"page":{"number":%d,"size":%d,"last":%d}}}`,
		b.String(), total, number, size, last,
	)
}

// TestAllProviderDrainsBeyondOnePage proves world.Processor.AllProvider (via
// requests.DrainProvider) fetches every page of the worlds list rather than
// stopping after the first response. atlas-world's GET /worlds is now
// paginated (task-117); the login server-list/world-select screens are a
// genuine startup consumer that must see every world. The fixture server
// serves 260 worlds across two pages of 250 - only a genuine drain picks up
// the world that lives on page 2.
func TestAllProviderDrainsBeyondOnePage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		number, _ := strconv.Atoi(r.URL.Query().Get("page[number]"))
		w.Header().Set("Content-Type", "application/vnd.api+json")
		if number == 2 {
			_, _ = w.Write([]byte(worldDoc(251, 260, 260, 2, 250, 2)))
			return
		}
		_, _ = w.Write([]byte(worldDoc(1, 250, 260, 1, 250, 2)))
	}))
	defer srv.Close()
	t.Setenv("WORLDS_SERVICE_URL", srv.URL+"/")

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), ten)
	l, _ := test.NewNullLogger()

	ws, err := world.NewProcessor(l, ctx).GetAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(ws) != 260 {
		t.Fatalf("expected 260 worlds (full drain), got %d; a single-fetch implementation would return 250", len(ws))
	}

	foundLast := false
	for _, w := range ws {
		if w.Name() == "World-260" {
			foundLast = true
		}
	}
	if !foundLast {
		t.Fatal("World-260 (page 2) must be present; single-fetch impl would miss it")
	}
}
