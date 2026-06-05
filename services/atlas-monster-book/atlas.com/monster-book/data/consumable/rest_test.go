package consumable

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
)

func newTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	return tm
}

// Proves the JSON:API stubs let Unmarshal succeed even with a relationships
// block present, and that unknown attributes are ignored.
func TestRestModel_UnmarshalWithRelationships(t *testing.T) {
	body := []byte(`{
		"data": {
			"type": "consumables",
			"id": "2380000",
			"attributes": {
				"monsterBook": true,
				"monsterId": 100100,
				"slotMax": 1,
				"price": 0
			},
			"relationships": {
				"rewards": { "data": [] }
			}
		}
	}`)
	var rm RestModel
	if err := jsonapi.Unmarshal(body, &rm); err != nil {
		t.Fatalf("jsonapi.Unmarshal: %v", err)
	}
	if rm.Id != 2380000 {
		t.Errorf("Id = %d, want 2380000", rm.Id)
	}
	if !rm.MonsterBook {
		t.Errorf("MonsterBook = false, want true")
	}
	if rm.MonsterId != 100100 {
		t.Errorf("MonsterId = %d, want 100100", rm.MonsterId)
	}
}

func TestGetById_RoundTrip(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/data/consumables/2380000") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{
			"data": {
				"type": "consumables",
				"id": "2380000",
				"attributes": { "monsterBook": true, "monsterId": 100100 },
				"relationships": { "rewards": { "data": [] } }
			}
		}`))
	}))
	defer srv.Close()
	defer SetBaseURLForTest(srv.URL)()

	ctx := tenant.WithContext(context.Background(), newTestTenant(t))
	m, err := NewProcessor(logrus.New(), ctx).GetById(2380000)
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if !m.MonsterBook() || m.MonsterId() != 100100 {
		t.Fatalf("model = %+v, want monsterBook=true monsterId=100100", m)
	}
}

func TestGetById_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	defer SetBaseURLForTest(srv.URL)()

	ctx := tenant.WithContext(context.Background(), newTestTenant(t))
	_, err := NewProcessor(logrus.New(), ctx).GetById(2380000)
	if err == nil {
		t.Fatal("expected error on 404, got nil")
	}
	if !errors.Is(err, requests.ErrNotFound) {
		t.Fatalf("expected requests.ErrNotFound, got %T: %v", err, err)
	}
}
