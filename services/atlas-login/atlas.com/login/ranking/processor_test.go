package ranking

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// rankingsFixture renders a bulk /rankings/characters JSON:API document for
// two characters with distinct, non-overlapping rank/rankMove/jobRank/
// jobRankMove/computedAt values so a field transposition in Extract would
// be caught. Character 20's rankMove/jobRankMove are negative to prove the
// signed int32 attribute survives the JSON round trip rather than being
// clamped or reinterpreted as unsigned.
const rankingsFixture = `{
  "data": [
    {
      "type": "rankings",
      "id": "10",
      "attributes": {
        "worldId": 0,
        "rank": 3,
        "rankMove": 1,
        "jobRank": 2,
        "jobRankMove": 4,
        "computedAt": "2026-07-17T00:00:00Z"
      }
    },
    {
      "type": "rankings",
      "id": "20",
      "attributes": {
        "worldId": 1,
        "rank": 55,
        "rankMove": -7,
        "jobRank": 12,
        "jobRankMove": -3,
        "computedAt": "2026-07-16T00:00:00Z"
      }
    }
  ]
}`

func TestGetByCharacterIdsDecodesRankings(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rankings/characters" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("ids"); got != "10,20" {
			t.Errorf("unexpected ids query %q", got)
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(rankingsFixture))
	}))
	defer srv.Close()
	t.Setenv("RANKINGS_SERVICE_URL", srv.URL+"/")

	tm, err := tenant.Register(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	ctx := tenant.WithContext(context.Background(), tm)

	rs, err := NewProcessor(logrus.New(), ctx).GetByCharacterIds([]uint32{10, 20})
	if err != nil {
		t.Fatalf("GetByCharacterIds failed: %v", err)
	}
	if len(rs) != 2 {
		t.Fatalf("expected 2 rankings, got %d", len(rs))
	}

	byId := make(map[uint32]Model, len(rs))
	for _, r := range rs {
		byId[r.CharacterId()] = r
	}

	a, ok := byId[10]
	if !ok {
		t.Fatalf("ranking for character 10 missing from decoded list")
	}
	if a.Rank() != 3 || a.RankMove() != 1 || a.JobRank() != 2 || a.JobRankMove() != 4 {
		t.Fatalf("character 10 decoded wrong: %+v", a)
	}

	b, ok := byId[20]
	if !ok {
		t.Fatalf("ranking for character 20 missing from decoded list")
	}
	if b.Rank() != 55 {
		t.Fatalf("character 20 rank decoded wrong: got %d want 55", b.Rank())
	}
	// The negative rankMove/jobRankMove must survive as the correct signed
	// int32, not wrap to a large unsigned value or get clamped to zero.
	if b.RankMove() != -7 {
		t.Fatalf("character 20 rankMove sign lost: got %d want -7", b.RankMove())
	}
	if b.JobRank() != 12 {
		t.Fatalf("character 20 jobRank decoded wrong: got %d want 12", b.JobRank())
	}
	if b.JobRankMove() != -3 {
		t.Fatalf("character 20 jobRankMove sign lost: got %d want -3", b.JobRankMove())
	}
}
