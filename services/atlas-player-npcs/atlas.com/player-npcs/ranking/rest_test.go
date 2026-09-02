package ranking

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func testCtx(t *testing.T) context.Context {
	t.Helper()
	tm, err := tenant.Register(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	return tenant.WithContext(context.Background(), tm)
}

// TestRestModel_Decode asserts the JSON:API decode of a captured
// rankings/characters/{id} payload: rank -> Rank(), jobRank -> JobRank().
func TestRestModel_Decode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"type":"rankings","id":"1001","attributes":{"rank":42,"jobRank":7}}}`))
	}))
	defer srv.Close()
	t.Setenv("RANKINGS_SERVICE_URL", srv.URL+"/")

	m, err := NewProcessor(logrus.New(), testCtx(t)).GetByCharacterId(1001, world.Id(0))
	if err != nil {
		t.Fatalf("GetByCharacterId returned error: %v", err)
	}
	if m.Rank() != 42 {
		t.Errorf("Rank() = %d, want 42", m.Rank())
	}
	if m.JobRank() != 7 {
		t.Errorf("JobRank() = %d, want 7", m.JobRank())
	}
}

// TestGetByCharacterId_AbsentRanking asserts that a 404 (a character with
// no computed ranking) yields a zero-rank result, not an error — such a
// character must still deploy.
func TestGetByCharacterId_AbsentRanking(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	t.Setenv("RANKINGS_SERVICE_URL", srv.URL+"/")

	m, err := NewProcessor(logrus.New(), testCtx(t)).GetByCharacterId(1002, world.Id(0))
	if err != nil {
		t.Fatalf("GetByCharacterId returned error: %v, want nil", err)
	}
	if m.Rank() != 0 || m.JobRank() != 0 {
		t.Errorf("m = %+v, want zero-value", m)
	}
}
