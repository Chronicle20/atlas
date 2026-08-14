package cash

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func TestGetByIdReadsAddTimeAndMaxDays(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"type":"cash_items","id":"5500001","attributes":{"addTime":604800,"maxDays":30}}}`))
	}))
	defer srv.Close()
	t.Setenv("DATA_SERVICE_URL", srv.URL+"/")

	l, _ := test.NewNullLogger()
	te, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), te)

	m, err := NewProcessor(l, ctx).GetById(5500001)
	if err != nil {
		t.Fatal(err)
	}
	if m.AddTime() != 604800 {
		t.Errorf("AddTime = %d, want 604800", m.AddTime())
	}
	if m.MaxDays() != 30 {
		t.Errorf("MaxDays = %d, want 30", m.MaxDays())
	}
}

// TestGetByIdReadsLife verifies that a cash item's info/life attribute (in
// DAYS) is decoded onto Model.Life() — the lifespan a Water of Life grants a
// revived pet, independent of the maxDays/addTime extender fields.
func TestGetByIdReadsLife(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"type":"cash_items","id":"5180000","attributes":{"life":90}}}`))
	}))
	defer srv.Close()
	t.Setenv("DATA_SERVICE_URL", srv.URL+"/")

	l, _ := test.NewNullLogger()
	te, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), te)

	m, err := NewProcessor(l, ctx).GetById(5180000)
	if err != nil {
		t.Fatal(err)
	}
	if m.Life() != 90 {
		t.Errorf("Life = %d, want 90", m.Life())
	}
}
