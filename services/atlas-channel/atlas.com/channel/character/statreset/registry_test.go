package statreset

import (
	"testing"
	"time"

	"github.com/google/uuid"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func mkTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	return tm
}

// TestAllow_FirstNudgePasses — recovery latency must be one packet, not one
// window. A character with no entry (fresh pod, reconnect, or a quiet period)
// is always honoured immediately.
func TestAllow_FirstNudgePasses(t *testing.T) {
	r := GetRegistry()
	tm := mkTenant(t)
	now := time.Unix(1_700_000_000, 0)

	if !r.Allow(tm, 1001, now) {
		t.Error("first nudge must pass")
	}
}

// TestAllow_ThrottlesInsideWindow reproduces the live v83 send rate: the
// client never advances its own m_tLastStatResetRequest anchor on v72–v87, so
// it emits once per frame (~30/s) indefinitely. The server bound must be
// independent of the client's advisory 200ms floor.
func TestAllow_ThrottlesInsideWindow(t *testing.T) {
	r := GetRegistry()
	tm := mkTenant(t)
	now := time.Unix(1_700_000_100, 0)

	if !r.Allow(tm, 1002, now) {
		t.Fatal("first nudge must pass")
	}
	honoured := 1
	// 30 nudges/second for one second, the measured unthrottled rate.
	for i := 1; i <= 30; i++ {
		if r.Allow(tm, 1002, now.Add(time.Duration(i)*33*time.Millisecond)) {
			honoured++
		}
	}
	if honoured != 1 {
		t.Errorf("honoured %d nudges inside a %v window, want 1", honoured, Window)
	}
}

// TestAllow_PassesAfterWindow — one honoured nudge per window, so a genuinely
// stuck client still recovers 10x faster than the 10s fleet sweep.
func TestAllow_PassesAfterWindow(t *testing.T) {
	r := GetRegistry()
	tm := mkTenant(t)
	now := time.Unix(1_700_000_200, 0)

	if !r.Allow(tm, 1003, now) {
		t.Fatal("first nudge must pass")
	}
	if r.Allow(tm, 1003, now.Add(Window-time.Millisecond)) {
		t.Error("nudge just inside the window must be dropped")
	}
	if !r.Allow(tm, 1003, now.Add(Window)) {
		t.Error("nudge at the window boundary must pass")
	}
}

// TestAllow_IsolatesCharactersAndTenants — the key is (tenant, character);
// one wedged client must not throttle anyone else.
func TestAllow_IsolatesCharactersAndTenants(t *testing.T) {
	r := GetRegistry()
	tmA := mkTenant(t)
	tmB := mkTenant(t)
	now := time.Unix(1_700_000_300, 0)

	if !r.Allow(tmA, 1004, now) {
		t.Fatal("tenant A character 1004 first nudge must pass")
	}
	if !r.Allow(tmA, 1005, now) {
		t.Error("a different character must not share throttle state")
	}
	if !r.Allow(tmB, 1004, now) {
		t.Error("the same character id under a different tenant must not share throttle state")
	}
}

// TestClearCharacter_ResetsState — called from the socket destroyer; without
// it the map leaks one entry per character ever seen by the pod.
func TestClearCharacter_ResetsState(t *testing.T) {
	r := GetRegistry()
	tm := mkTenant(t)
	now := time.Unix(1_700_000_400, 0)

	if !r.Allow(tm, 1006, now) {
		t.Fatal("first nudge must pass")
	}
	if r.Allow(tm, 1006, now.Add(10*time.Millisecond)) {
		t.Fatal("second nudge inside the window must be dropped")
	}

	r.ClearCharacter(tm, 1006)

	if !r.Allow(tm, 1006, now.Add(20*time.Millisecond)) {
		t.Error("after ClearCharacter the next nudge must pass")
	}
}
