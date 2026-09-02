package controller

import (
	"atlas-channel/playernpc"
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	npcpkt "github.com/Chronicle20/atlas/libs/atlas-packet/npc"
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

// playerNpcJSON is one deployed Player NPC resource, with the position
// fields the grant payload carries set to distinguishable values.
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
			"x": 111,
			"cy": 222,
			"fh": 3,
			"rx0": 44,
			"rx1": 155,
			"dir": 1,
			"worldRank": 1,
			"overallRank": 1,
			"worldJobRank": 1,
			"overallJobRank": 1,
			"equipment": [],
			"deployedAt": "2024-01-01T00:00:00Z"
		}
	}`, uuid.New().String(), scriptId, objectId)
}

func announceTestCtx(t *testing.T) (context.Context, field.Model) {
	t.Helper()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	return tenant.WithContext(context.Background(), ten), field.NewBuilder(0, 1, 100000000).Build()
}

// TestGrantBodyResolvesPlayerNpcThroughPlayerNpcs covers the Player NPC arm
// of grantBody's oid-band dispatch (design D-5): the object is not in
// atlas-data's per-map life list, so the payload must come from
// atlas-player-npcs, carrying that model's own position.
func TestGrantBodyResolvesPlayerNpcThroughPlayerNpcs(t *testing.T) {
	const scriptId uint32 = 9901000
	const objectId uint32 = 101000

	l, _ := test.NewNullLogger()
	ctx, f := announceTestCtx(t)
	stubPlayerNpcList(t, playerNpcJSON(objectId, scriptId))

	enc, err := grantBody(l, ctx, f, objectId)
	if err != nil {
		t.Fatalf("grantBody: %v", err)
	}
	opts := map[string]interface{}{}
	got := enc(l, ctx)(opts)
	want := npcpkt.NpcControllerGrantBody(objectId, scriptId, 111, 222, 1, 3, 44, 155, true)(l, ctx)(opts)
	if !bytes.Equal(got, want) {
		t.Fatalf("grant body = %v, want %v", got, want)
	}
}

// TestGrantBodyPlayerNpcNotDeployed asserts the branch surfaces the read
// client's not-found rather than falling through to the atlas-data lookup,
// which can never resolve an oid in the reserved band.
func TestGrantBodyPlayerNpcNotDeployed(t *testing.T) {
	l, _ := test.NewNullLogger()
	ctx, f := announceTestCtx(t)
	stubPlayerNpcList(t)

	if _, err := grantBody(l, ctx, f, 101000); err != playernpc.ErrNotFound {
		t.Fatalf("grantBody err = %v, want %v", err, playernpc.ErrNotFound)
	}
}

// TestPlayerNpcGrantBodyMatchesResolvedBody pins the caller-supplied-model
// path (AnnounceGrantWith) to the resolved one: a caller holding the model
// must produce byte-identical payload to grantBody's read.
func TestPlayerNpcGrantBodyMatchesResolvedBody(t *testing.T) {
	const scriptId uint32 = 9901000
	const objectId uint32 = 101000

	l, _ := test.NewNullLogger()
	ctx, f := announceTestCtx(t)
	stubPlayerNpcList(t, playerNpcJSON(objectId, scriptId))

	n, err := playernpc.NewProcessor(l, ctx).GetInMapByObjectId(f, objectId)
	if err != nil {
		t.Fatalf("GetInMapByObjectId: %v", err)
	}
	resolved, err := grantBody(l, ctx, f, objectId)
	if err != nil {
		t.Fatalf("grantBody: %v", err)
	}
	opts := map[string]interface{}{}
	if !bytes.Equal(PlayerNpcGrantBody(n)(l, ctx)(opts), resolved(l, ctx)(opts)) {
		t.Fatal("PlayerNpcGrantBody diverges from grantBody's resolved payload")
	}
}
