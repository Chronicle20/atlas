package character

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

const charactersFixture = `{
  "data": [
    {
      "type": "characters",
      "id": "1",
      "attributes": {
        "accountId": 1000,
        "worldId": 0,
        "name": "Alpha",
        "level": 50,
        "experience": 1234,
        "jobId": 112,
        "gm": 0
      },
      "relationships": {
        "equipment": {"data": []},
        "inventories": {"data": []}
      }
    },
    {
      "type": "characters",
      "id": "2",
      "attributes": {
        "accountId": 1001,
        "worldId": 1,
        "name": "Beta",
        "level": 30,
        "experience": 55,
        "jobId": 0,
        "gm": 1
      },
      "relationships": {
        "equipment": {"data": []},
        "inventories": {"data": []}
      }
    }
  ]
}`

func TestGetAllDecodesTrimmedModel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/characters" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(charactersFixture))
	}))
	defer srv.Close()
	t.Setenv("CHARACTERS_SERVICE_URL", srv.URL+"/")

	tm, err := tenant.Register(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	ctx := tenant.WithContext(context.Background(), tm)

	cs, err := NewProcessor(logrus.New(), ctx).GetAll()
	if err != nil {
		t.Fatalf("GetAll failed: %v", err)
	}
	if len(cs) != 2 {
		t.Fatalf("expected 2 characters, got %d", len(cs))
	}
	if cs[0].Id() != 1 || cs[0].Level() != 50 || cs[0].Experience() != 1234 || cs[0].JobId() != 112 || cs[0].Gm() != 0 {
		t.Fatalf("character 1 decoded wrong: %+v", cs[0])
	}
	if cs[1].WorldId() != 1 || cs[1].Gm() != 1 {
		t.Fatalf("character 2 decoded wrong: %+v", cs[1])
	}
}

// characterDoc renders a JSON:API document for characters [from, to],
// including the pagination envelope atlas-character actually emits
// (server.MarshalPaginatedResponse -> paginate.Envelope.Meta(), see
// libs/atlas-rest/server/paginate/envelope.go). Each character gets a
// distinct id/accountId/level/experience derived from its index so the
// test can assert identity of specific page-1 and page-2 items. The
// relationships block is kept on every item — omitting it would mask the
// jsonapi-stub failure mode the single-page fixture guards against.
func characterDoc(from, to, total, number, size, last int) string {
	var b strings.Builder
	for i := from; i <= to; i++ {
		if b.Len() > 0 {
			b.WriteString(",")
		}
		b.WriteString(fmt.Sprintf(
			`{"type":"characters","id":"%d","attributes":{"accountId":%d,"worldId":2,"name":"Drain-%d","level":%d,"experience":%d,"jobId":200,"gm":0},"relationships":{"equipment":{"data":[]},"inventories":{"data":[]}}}`,
			i, 9000+i, i, (i%100)+1, i*10,
		))
	}
	return fmt.Sprintf(
		`{"data":[%s],"meta":{"total":%d,"page":{"number":%d,"size":%d,"last":%d}}}`,
		b.String(), total, number, size, last,
	)
}

// TestAllProviderDrainsBeyondOnePage proves character.Processor.AllProvider
// (via requests.DrainProvider) fetches every page of the characters list
// rather than stopping after the first response. Ranking computation is a
// genuine semantic-"all" consumer: a tenant with 251+ characters must have
// every one of them ranked, not just the first page[size]=250. The fixture
// server serves 251 characters across two pages of 250 — only a genuine
// drain picks up the character on page 2. If DrainProvider stopped after
// page 1 (e.g. AllProvider reverted to a plain single-page GetRequest),
// this test would see 250 characters and fail both the count and the
// page-2-identity assertions below.
func TestAllProviderDrainsBeyondOnePage(t *testing.T) {
	const total = 251
	const pageSize = 250

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		number, _ := strconv.Atoi(r.URL.Query().Get("page[number]"))
		w.Header().Set("Content-Type", "application/vnd.api+json")
		if number == 2 {
			_, _ = w.Write([]byte(characterDoc(251, 251, total, 2, pageSize, 2)))
			return
		}
		_, _ = w.Write([]byte(characterDoc(1, 250, total, 1, pageSize, 2)))
	}))
	defer srv.Close()
	t.Setenv("CHARACTERS_SERVICE_URL", srv.URL+"/")

	tm, err := tenant.Register(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	ctx := tenant.WithContext(context.Background(), tm)

	cs, err := NewProcessor(logrus.New(), ctx).GetAll()
	if err != nil {
		t.Fatalf("GetAll failed: %v", err)
	}
	if len(cs) != total {
		t.Fatalf("expected %d characters (full drain), got %d; a single-fetch implementation would return %d", total, len(cs), pageSize)
	}

	var first, last *Model
	for i := range cs {
		switch cs[i].Id() {
		case 1:
			first = &cs[i]
		case 251:
			last = &cs[i]
		}
	}
	if first == nil {
		t.Fatal("character 1 (page 1) must be present")
	}
	if first.Level() != 2 || first.Experience() != 10 {
		t.Fatalf("character 1 decoded wrong: %+v", first)
	}
	if last == nil {
		t.Fatal("character 251 (page 2) must be present; single-fetch impl would miss it")
	}
	if last.Level() != 52 || last.Experience() != 2510 {
		t.Fatalf("character 251 decoded wrong: %+v", last)
	}
}
