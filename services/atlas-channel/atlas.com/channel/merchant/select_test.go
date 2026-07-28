package merchant

import (
	"testing"

	"github.com/google/uuid"
)

// A character accumulates closed shop rows across a play session (each open →
// close leaves a StateClosed leftover). The owl warp resolves the target by
// characterId, so it must pick the one live shop, not shops[0]. Regression:
// char 2 had three StateClosed leftovers ahead of no/one open shop, so the warp
// ladder read State=Closed and answered CLOSED for a store the search had just
// surfaced (task-127).
func TestSelectOpenShop(t *testing.T) {
	open := Model{id: uuid.New(), state: StateOpen, mapId: 910000001}
	closed1 := Model{id: uuid.New(), state: StateClosed, mapId: 910000001}
	closed2 := Model{id: uuid.New(), state: StateClosed, mapId: 910000001}
	maint := Model{id: uuid.New(), state: StateMaintenance, mapId: 910000001}

	t.Run("prefers open over leading closed leftovers", func(t *testing.T) {
		got, ok := SelectOpenShop([]Model{closed1, closed2, open})
		if !ok || got.Id() != open.Id() {
			t.Fatalf("want open shop %v, got ok=%v id=%v", open.Id(), ok, got.Id())
		}
	})
	t.Run("all closed -> not found", func(t *testing.T) {
		if _, ok := SelectOpenShop([]Model{closed1, closed2}); ok {
			t.Fatalf("want not found when every shop is closed")
		}
	})
	t.Run("maintenance surfaces so the ladder reports it faithfully", func(t *testing.T) {
		got, ok := SelectOpenShop([]Model{closed1, maint})
		if !ok || got.State() != StateMaintenance {
			t.Fatalf("want maintenance shop, got ok=%v state=%d", ok, got.State())
		}
	})
	t.Run("open wins over maintenance", func(t *testing.T) {
		got, ok := SelectOpenShop([]Model{maint, open})
		if !ok || got.State() != StateOpen {
			t.Fatalf("want open shop, got ok=%v state=%d", ok, got.State())
		}
	})
	t.Run("empty -> not found", func(t *testing.T) {
		if _, ok := SelectOpenShop(nil); ok {
			t.Fatalf("want not found for empty input")
		}
	})
}
