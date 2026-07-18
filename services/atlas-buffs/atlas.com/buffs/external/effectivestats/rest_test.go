package effectivestats

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
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

// TestRequestByCharacter_RoundTrip stands up an httptest server returning a
// realistic JSON:API document for an effective-stats resource, INCLUDING a
// relationships block, and drives it through the real requests.GetRequest
// decode path. This proves the SetToOneReferenceID/SetToManyReferenceIDs
// stubs added per libs/atlas-rest/CLAUDE.md let api2go decode a response
// that carries relationships.
//
// It also pins the load-bearing "maxHP" (uppercase HP) JSON tag: a
// lowercase drift here would silently decode MaxHp as 0 and make every
// Dark Knight read as zero max HP for berserk evaluation.
func TestRequestByCharacter_RoundTrip(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/worlds/0/channels/1/characters/42/stats") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{
			"data": {
				"type": "effective-stats",
				"id": "42",
				"attributes": {
					"maxHP": 12345
				},
				"relationships": {
					"character": {
						"data": {"type": "characters", "id": "42"}
					},
					"equipment": {
						"data": [
							{"type": "equipment", "id": "1"}
						]
					}
				}
			}
		}`))
	}))
	defer srv.Close()

	t.Setenv("EFFECTIVE_STATS_SERVICE_URL", srv.URL+"/")

	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)

	rm, err := RequestByCharacter(world.Id(0), channel.Id(1), 42)(logrus.New(), ctx)
	if err != nil {
		t.Fatalf("RequestByCharacter: %v", err)
	}
	if rm.Id != "42" {
		t.Errorf("Id = %q, want %q", rm.Id, "42")
	}
	if rm.MaxHp != 12345 {
		t.Errorf("MaxHp = %d, want 12345 (maxHP attribute)", rm.MaxHp)
	}
}
