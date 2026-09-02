package npc

import (
	"atlas-channel/playernpc"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// stubPlayerNpcList stands up a fake atlas-player-npcs list endpoint and
// points playernpc's base URL at it for the test's duration.
func stubPlayerNpcList(t *testing.T, entries ...string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = fmt.Fprintf(w, `{"data":[%s]}`, strings.Join(entries, ","))
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(playernpc.SetBaseURLForTest(srv.URL))
}

func playerNpcJSON(objectId uint32, scriptId uint32) string {
	return fmt.Sprintf(`{
		"type": "player-npcs",
		"id": "%s",
		"attributes": {
			"characterId": 1,
			"name": "Statue",
			"worldId": 0,
			"mapId": 100000000,
			"scriptId": %d,
			"objectId": %d,
			"gender": 0,
			"skin": 0,
			"face": 20000,
			"hair": 30000,
			"jobId": 100,
			"x": 100,
			"cy": 200,
			"fh": 1,
			"rx0": 50,
			"rx1": 150,
			"dir": 0,
			"worldRank": 1,
			"overallRank": 1,
			"worldJobRank": 1,
			"overallJobRank": 1,
			"equipment": [],
			"deployedAt": "2024-01-01T00:00:00Z"
		}
	}`, uuid.New().String(), scriptId, objectId)
}

func resolveTestCtx(t *testing.T) (context.Context, field.Model) {
	t.Helper()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	return tenant.WithContext(context.Background(), ten), field.NewBuilder(0, 1, 100000000).Build()
}

// TestResolveTemplateResolvesPlayerNpc covers the Player NPC arm of the
// inbound action/move resolver: the oid is in the reserved band, so it must
// resolve through atlas-player-npcs -- atlas-data's per-map life list can
// never contain it (task-251 bug report §5 review, blocking #1).
func TestResolveTemplateResolvesPlayerNpc(t *testing.T) {
	const scriptId uint32 = 9901000
	const objectId uint32 = 101000

	l, _ := test.NewNullLogger()
	ctx, f := resolveTestCtx(t)
	stubPlayerNpcList(t, playerNpcJSON(objectId, scriptId))

	got, err := ResolveTemplate(l, ctx, f, objectId)
	if err != nil {
		t.Fatalf("ResolveTemplate: %v", err)
	}
	if got != scriptId {
		t.Fatalf("ResolveTemplate = %d, want %d", got, scriptId)
	}
}

// TestResolveTemplatePlayerNpcNotDeployed asserts an oid in the band that is
// not deployed is rejected rather than relayed.
func TestResolveTemplatePlayerNpcNotDeployed(t *testing.T) {
	l, _ := test.NewNullLogger()
	ctx, f := resolveTestCtx(t)
	stubPlayerNpcList(t)

	if _, err := ResolveTemplate(l, ctx, f, 101000); err != playernpc.ErrNotFound {
		t.Fatalf("ResolveTemplate err = %v, want %v", err, playernpc.ErrNotFound)
	}
}
