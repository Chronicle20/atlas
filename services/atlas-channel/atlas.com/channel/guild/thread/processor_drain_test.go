package thread_test

import (
	"atlas-channel/guild/thread"
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

// threadsDoc renders a JSON:API "threads" document for thread ids [from, to].
// meta describes the current page/total so requests.DrainProvider can
// decide whether to keep paging.
func threadsDoc(from, to int, total, number, size, last int) string {
	var b strings.Builder
	for id := from; id <= to; id++ {
		if b.Len() > 0 {
			b.WriteString(",")
		}
		b.WriteString(fmt.Sprintf(
			`{"id":"%d","type":"threads","attributes":{"posterId":100,"title":"Thread %d","message":"message","emoticonId":0,"notice":false,"replies":[],"createdAt":"2024-01-01T00:00:00Z"}}`,
			id, id,
		))
	}
	return fmt.Sprintf(
		`{"data":[%s],"meta":{"total":%d,"page":{"number":%d,"size":%d,"last":%d}}}`,
		b.String(), total, number, size, last,
	)
}

func threadDrainTestServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		number, _ := strconv.Atoi(r.URL.Query().Get("page[number]"))
		w.Header().Set("Content-Type", "application/vnd.api+json")
		if number == 2 {
			_, _ = w.Write([]byte(threadsDoc(251, 300, 300, 2, 250, 2)))
			return
		}
		_, _ = w.Write([]byte(threadsDoc(1, 250, 300, 1, 250, 2)))
	}))
}

// TestAllModelProviderDrainsBeyondOnePage proves AllModelProvider (via
// requests.DrainProvider) fetches every page of a guild's thread log rather
// than stopping after the first response. atlas-guilds' GET
// /guilds/{guildId}/threads is now paginated (task-117); the guild BBS list
// display (socket/handler/guild_bbs.go) is a hot per-character-request path
// that must see the whole log -- the fixture server serves 300 threads
// across two pages of 250, so only a genuine drain picks up thread id 300,
// which lives on page 2.
func TestAllModelProviderDrainsBeyondOnePage(t *testing.T) {
	srv := threadDrainTestServer()
	defer srv.Close()
	t.Setenv("GUILD_THREADS_SERVICE_URL", srv.URL+"/")

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), ten)
	l, _ := test.NewNullLogger()

	ms, err := thread.NewProcessor(l, ctx).GetAll(1)
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 300 {
		t.Fatalf("expected 300 threads (full drain), got %d; a single-fetch implementation would return 250", len(ms))
	}

	foundLast := false
	for _, m := range ms {
		if m.Id() == 300 {
			foundLast = true
		}
	}
	if !foundLast {
		t.Fatal("thread id 300 (page 2) must be present; single-fetch impl would miss it")
	}
}
