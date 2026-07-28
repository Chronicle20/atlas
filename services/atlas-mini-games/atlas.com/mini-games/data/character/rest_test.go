package character

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func newTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	return tm
}

// TestHp_Served drives a real JSON:API character document through the api2go
// unmarshal path and asserts Hp returns the served value (EXT-02).
func TestHp_Served(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/characters/12345") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{
			"data": {
				"type": "characters",
				"id": "12345",
				"attributes": {
					"name": "Hero",
					"hp": 4200
				}
			}
		}`))
	}))
	defer srv.Close()
	defer SetBaseURLForTest(srv.URL)()

	ctx := tenant.WithContext(context.Background(), newTestTenant(t))
	hp, err := NewProcessor(logrus.New(), ctx).Hp(12345)
	if err != nil {
		t.Fatalf("Hp: %v", err)
	}
	if hp != 4200 {
		t.Fatalf("hp: want 4200, got %d", hp)
	}
}
