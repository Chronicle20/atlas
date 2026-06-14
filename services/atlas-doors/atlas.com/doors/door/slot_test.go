package door

import "testing"

func TestComputeSlotSolo(t *testing.T) {
	if got := ComputeSlot(0, []uint32{}, 42); got != 0 {
		t.Fatalf("solo slot want 0 got %d", got)
	}
}

func TestComputeSlotPartyIndex(t *testing.T) {
	members := []uint32{10, 20, 30}
	if got := ComputeSlot(7, members, 30); got != 2 {
		t.Fatalf("want slot 2 got %d", got)
	}
	if got := ComputeSlot(7, members, 10); got != 0 {
		t.Fatalf("want slot 0 got %d", got)
	}
}

func TestComputeSlotNotMemberFallsToZero(t *testing.T) {
	if got := ComputeSlot(7, []uint32{10, 20}, 99); got != 0 {
		t.Fatalf("non-member want 0 got %d", got)
	}
}

func TestResolveTownPortalWithEnoughDoorPortals(t *testing.T) {
	portals := []TownPortal{{X: -10, Y: 1}, {X: -20, Y: 2}, {X: -30, Y: 3},
		{X: -40, Y: 4}, {X: -50, Y: 5}, {X: -60, Y: 6}}
	wireId, x, y, ok := ResolveTownPortal(portals, 3, defaultTownX, defaultTownY)
	if !ok || wireId != 0x83 || x != -40 || y != 4 {
		t.Fatalf("want 0x83/-40/4 got %d/%d/%d ok=%v", wireId, x, y, ok)
	}
}

func TestResolveTownPortalFallbackWhenTooFew(t *testing.T) {
	portals := []TownPortal{{X: -10, Y: 1}} // only 1 door portal
	wireId, x, y, ok := ResolveTownPortal(portals, 3, 7, 8)
	// wire id still 0x80+slot; position falls back to provided default
	if !ok || wireId != 0x83 || x != 7 || y != 8 {
		t.Fatalf("fallback wrong: %d/%d/%d ok=%v", wireId, x, y, ok)
	}
}
