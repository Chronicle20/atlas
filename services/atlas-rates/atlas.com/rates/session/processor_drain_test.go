package session

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
)

// sessionDoc renders a JSON:API page of sessions [from, to). Each session
// spans exactly one minute, from base+i minutes to base+i minutes+1 minute.
// meta describes the current page/total so requests.DrainProvider can decide
// whether to keep paging.
func sessionDoc(characterId uint32, base time.Time, from, to, total, number, size, last int) string {
	var b strings.Builder
	for i := from; i < to; i++ {
		if b.Len() > 0 {
			b.WriteString(",")
		}
		login := base.Add(time.Duration(i) * time.Minute)
		logout := login.Add(time.Minute)
		b.WriteString(fmt.Sprintf(
			`{"id":"%d","type":"sessions","attributes":{"characterId":%d,"worldId":0,"channelId":0,"loginTime":%q,"logoutTime":%q}}`,
			i, characterId, login.Format(time.RFC3339), logout.Format(time.RFC3339),
		))
	}
	return fmt.Sprintf(
		`{"data":[%s],"meta":{"total":%d,"page":{"number":%d,"size":%d,"last":%d}}}`,
		b.String(), total, number, size, last,
	)
}

// TestGetSessionsSinceDrainsAllPages proves GetSessionsSince (and therefore
// ComputePlaytimeInRange, which sums OverlapsWith over the returned slice)
// drains every page of the character's session history rather than stopping
// after page 1. The fixture serves 300 one-minute sessions across two pages
// of 250; a single-fetch implementation only sees the oldest 250 (page 1,
// login_time ASC per the real endpoint's ordering) and undercounts playtime
// by 50 minutes.
func TestGetSessionsSinceDrainsAllPages(t *testing.T) {
	const characterId = 12345
	const totalSessions = 300
	const pageSize = 250
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	var gotSince []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSince = append(gotSince, r.URL.Query().Get("since"))
		number, _ := strconv.Atoi(r.URL.Query().Get("page[number]"))
		w.Header().Set("Content-Type", "application/vnd.api+json")
		if number == 2 {
			_, _ = w.Write([]byte(sessionDoc(characterId, base, 250, 300, totalSessions, 2, pageSize, 2)))
			return
		}
		_, _ = w.Write([]byte(sessionDoc(characterId, base, 0, 250, totalSessions, 1, pageSize, 2)))
	}))
	defer srv.Close()
	t.Setenv("CHARACTER_SERVICE_URL", srv.URL+"/")

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), ten)
	l, _ := test.NewNullLogger()

	since := base.Add(-time.Hour)
	p := NewProcessor(l, ctx)
	sessions, err := p.GetSessionsSince(characterId, since)
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != totalSessions {
		t.Fatalf("GetSessionsSince returned %d sessions, want %d (page 2 dropped by a single-fetch implementation)", len(sessions), totalSessions)
	}

	// Every request must carry the since filter: DrainProvider must preserve
	// existing query params across every page it re-reads, not just page 1.
	if len(gotSince) < 2 {
		t.Fatalf("expected at least 2 requests (one per page), got %d", len(gotSince))
	}
	wantSince := strconv.FormatInt(since.Unix(), 10)
	for i, s := range gotSince {
		if s != wantSince {
			t.Fatalf("request %d: since=%q, want %q (since filter lost on a later page)", i, s, wantSince)
		}
	}

	// ComputePlaytimeInRange sums OverlapsWith across the whole drained
	// collection: 300 one-minute sessions covering [base, base+300min) should
	// yield exactly 300 minutes when the query range covers all of them. A
	// single-fetch implementation (page 1 only) would report 250 minutes.
	end := base.Add(time.Duration(totalSessions) * time.Minute)
	playtime, err := p.ComputePlaytimeInRange(characterId, since, end)
	if err != nil {
		t.Fatal(err)
	}
	wantPlaytime := time.Duration(totalSessions) * time.Minute
	if playtime != wantPlaytime {
		t.Fatalf("ComputePlaytimeInRange = %v, want %v (a single-fetch implementation would undercount to 250m)", playtime, wantPlaytime)
	}
}
