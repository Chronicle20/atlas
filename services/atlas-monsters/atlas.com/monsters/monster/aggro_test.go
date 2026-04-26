package monster

import "testing"

func TestIsAggroIdleBoundary(t *testing.T) {
	now := int64(20_000)

	// Exactly at threshold: not yet idle.
	e := entry{LastHitMs: now - AggroIdleThresholdMs}
	if IsAggroIdle(e, now) {
		t.Errorf("entry at exactly threshold (delta=%d) should NOT be idle", AggroIdleThresholdMs)
	}

	// Past threshold by 1 ms: idle.
	e = entry{LastHitMs: now - AggroIdleThresholdMs - 1}
	if !IsAggroIdle(e, now) {
		t.Errorf("entry past threshold by 1ms should be idle")
	}

	// Just-hit: not idle.
	e = entry{LastHitMs: now}
	if IsAggroIdle(e, now) {
		t.Errorf("just-hit entry should not be idle")
	}
}

func TestAggroConstants(t *testing.T) {
	if AggroIdleThresholdMs != 10_000 {
		t.Errorf("AggroIdleThresholdMs: expected 10000, got %d", AggroIdleThresholdMs)
	}
	if AggroDecayMultiplier != 0.85 {
		t.Errorf("AggroDecayMultiplier: expected 0.85, got %v", AggroDecayMultiplier)
	}
	if AggroDecayFloor != 1 {
		t.Errorf("AggroDecayFloor: expected 1, got %d", AggroDecayFloor)
	}
	if AggroSweepInterval.Milliseconds() != 1500 {
		t.Errorf("AggroSweepInterval: expected 1500ms, got %v", AggroSweepInterval)
	}
}
