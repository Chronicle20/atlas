package buff_test

import (
	"atlas-channel/character/buff"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// TestByCharacterIdProviderTreatsNotFoundAsNoBuffs proves a 404 from
// atlas-buffs' GET /characters/{characterId}/buffs is read as "this
// character has zero buffs", not as a fetch failure.
//
// atlas-buffs materializes a character's buff registry entry lazily: until
// something applies a buff, character.Processor.GetById returns ErrNotFound
// and the resource replies 404 (services/atlas-buffs/.../character/resource.go
// handleGetBuffs). Every caller here asks the same question -- "what buffs
// does this character have?" -- and for a character with none the honest
// answer is the empty set.
//
// Propagating the 404 as an error made callers that skip-on-error drop
// exactly the players who had no buffs yet. Observed live on atlas-pr-1227:
// Echo of Hero's map-wide fan-out ran its GM-hidden pre-check
// (skill/handler/echoofhero), got "not found" for a buffless recipient, and
// logged fetch_failures:1 / applied:0 -- the recipient never got the buff.
// The same skip-on-error seam exists in healdispel, hide, and the
// map-consumer spawn paths, so the fix belongs here at the source.
func TestByCharacterIdProviderTreatsNotFoundAsNoBuffs(t *testing.T) {
	characterId := uint32(2)

	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	t.Setenv("BUFFS_SERVICE_URL", srv.URL+"/")

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), ten)
	l, _ := test.NewNullLogger()

	ms, err := buff.NewProcessor(l, ctx).GetByCharacterId(characterId)
	if err != nil {
		t.Fatalf("404 (no buff registry entry) must read as zero buffs, got error: %v", err)
	}
	if len(ms) != 0 {
		t.Fatalf("expected zero buffs, got %d", len(ms))
	}
	if calls == 0 {
		t.Fatal("expected the provider to actually issue the request")
	}
}
