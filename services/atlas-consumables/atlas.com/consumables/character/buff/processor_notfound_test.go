package buff_test

import (
	"atlas-consumables/character/buff"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// TestGetByCharacterIdTreatsNotFoundAsNoBuffs proves a 404 from atlas-buffs'
// GET /characters/{characterId}/buffs is read as "this character has zero
// buffs", not as a fetch failure. atlas-buffs materializes a character's
// buff registry entry lazily, so the resource replies 404 until something
// applies a first buff -- that is "no buffs", not an error. See
// atlas-channel's ByCharacterIdProvider for the source of this behavior.
func TestGetByCharacterIdTreatsNotFoundAsNoBuffs(t *testing.T) {
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
