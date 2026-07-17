package channel_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"atlas-login/channel"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
)

// channelDoc renders a JSON:API document for channels [from, to] registered
// to a world. Each channel's port is unique so the test can assert presence
// of a page-2-only item.
func channelDoc(from, to int, total, number, size, last int) string {
	var b strings.Builder
	for id := from; id <= to; id++ {
		if b.Len() > 0 {
			b.WriteString(",")
		}
		b.WriteString(fmt.Sprintf(
			`{"id":"00000000-0000-0000-0000-%012d","type":"channels","attributes":{"worldId":1,"channelId":%d,"ipAddress":"10.0.0.1","port":%d,"currentCapacity":0,"maxCapacity":100,"createdAt":"2026-01-01T00:00:00Z"}}`,
			id, id%256, id,
		))
	}
	return fmt.Sprintf(
		`{"data":[%s],"meta":{"total":%d,"page":{"number":%d,"size":%d,"last":%d}}}`,
		b.String(), total, number, size, last,
	)
}

// TestByWorldModelProviderDrainsBeyondOnePage proves
// channel.Processor.ByWorldModelProvider (via requests.DrainProvider)
// fetches every page of a world's channel-server list rather than stopping
// after the first response. atlas-world's GET /worlds/{worldId}/channels is
// now paginated (task-117); GetRandomInWorld (used to route a logging-in
// character to a channel) is a genuine startup consumer that must see every
// channel. The fixture server serves 260 channels across two pages of 250 -
// only a genuine drain picks up the channel that lives on page 2.
func TestByWorldModelProviderDrainsBeyondOnePage(t *testing.T) {
	worldId := world.Id(1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		number, _ := strconv.Atoi(r.URL.Query().Get("page[number]"))
		w.Header().Set("Content-Type", "application/vnd.api+json")
		if number == 2 {
			_, _ = w.Write([]byte(channelDoc(251, 260, 260, 2, 250, 2)))
			return
		}
		_, _ = w.Write([]byte(channelDoc(1, 250, 260, 1, 250, 2)))
	}))
	defer srv.Close()
	t.Setenv("CHANNELS_SERVICE_URL", srv.URL+"/")

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), ten)
	l, _ := test.NewNullLogger()

	cs, err := channel.NewProcessor(l, ctx).GetForWorld(worldId)
	if err != nil {
		t.Fatal(err)
	}
	if len(cs) != 260 {
		t.Fatalf("expected 260 channels (full drain), got %d; a single-fetch implementation would return 250", len(cs))
	}

	foundLast := false
	for _, c := range cs {
		if c.Port() == 260 {
			foundLast = true
		}
	}
	if !foundLast {
		t.Fatal("channel with port 260 (page 2) must be present; single-fetch impl would miss it")
	}
}
