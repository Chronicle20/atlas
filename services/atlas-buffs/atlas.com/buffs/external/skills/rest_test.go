package skills

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
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

// TestRequestByCharacterAndSkill_RoundTrip stands up an httptest server
// returning a realistic JSON:API document for a character-skill resource,
// INCLUDING a relationships block, and drives it through the real
// requests.GetRequest decode path. This proves the SetToOneReferenceID/
// SetToManyReferenceIDs stubs added per libs/atlas-rest/CLAUDE.md let
// api2go decode a response that carries relationships, and pins the
// wire-tag mapping for level.
func TestRequestByCharacterAndSkill_RoundTrip(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/characters/42/skills/1320006") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{
			"data": {
				"type": "skills",
				"id": "1320006",
				"attributes": {
					"level": 30
				},
				"relationships": {
					"character": {
						"data": {"type": "characters", "id": "42"}
					},
					"cooldowns": {
						"data": []
					}
				}
			}
		}`))
	}))
	defer srv.Close()

	t.Setenv("SKILLS_SERVICE_URL", srv.URL+"/")

	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)

	rm, err := RequestByCharacterAndSkill(42, 1320006)(logrus.New(), ctx)
	if err != nil {
		t.Fatalf("RequestByCharacterAndSkill: %v", err)
	}
	if rm.Id != 1320006 {
		t.Errorf("Id = %d, want 1320006", rm.Id)
	}
	if rm.Level != 30 {
		t.Errorf("Level = %d, want 30", rm.Level)
	}
}
