package objectid

import "testing"

func TestPlayerNpcObjectIdBandDoesNotCollide(t *testing.T) {
	t.Run("above WZ NPC ids", func(t *testing.T) {
		if PlayerNpcObjectIdBase != uint32(100000) {
			t.Fatalf("expected PlayerNpcObjectIdBase == 100000, got %d", PlayerNpcObjectIdBase)
		}
	})

	t.Run("below the shared allocator", func(t *testing.T) {
		if !(PlayerNpcObjectIdBase < MinId) {
			t.Fatalf("expected PlayerNpcObjectIdBase (%d) < MinId (%d)", PlayerNpcObjectIdBase, MinId)
		}
	})

	t.Run("whole script range fits the band", func(t *testing.T) {
		for _, scriptId := range []uint32{9900000, 9901000, 9906599} {
			oid := PlayerNpcObjectIdFor(scriptId)
			if oid < PlayerNpcObjectIdBase || oid >= MinId {
				t.Fatalf("scriptId %d: PlayerNpcObjectIdFor = %d, expected within [%d, %d)", scriptId, oid, PlayerNpcObjectIdBase, MinId)
			}
		}
	})

	t.Run("lowest script id maps to the base", func(t *testing.T) {
		if got := PlayerNpcObjectIdFor(9900000); got != uint32(100000) {
			t.Fatalf("expected PlayerNpcObjectIdFor(9900000) == 100000, got %d", got)
		}
	})

	t.Run("highest maps below MinId", func(t *testing.T) {
		if got := PlayerNpcObjectIdFor(9906599); got != uint32(106599) {
			t.Fatalf("expected PlayerNpcObjectIdFor(9906599) == 106599, got %d", got)
		}
	})
}
