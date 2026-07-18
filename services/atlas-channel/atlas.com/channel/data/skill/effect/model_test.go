package effect

import "testing"

// bulletCount is the per-attack projectile count (e.g. Lucky Seven 2, Triple
// Throw 3); bulletConsume is the one-time Shadow Stars activation cost (200).
func TestModelBulletCount(t *testing.T) {
	m := Model{bulletCount: 2}
	if got := m.BulletCount(); got != 2 {
		t.Fatalf("BulletCount() = %d, want 2", got)
	}
}

func TestModelBulletConsume(t *testing.T) {
	m := Model{bulletConsume: 200}
	if got := m.BulletConsume(); got != 200 {
		t.Fatalf("BulletConsume() = %d, want 200", got)
	}
}

// TestExtractBulletFields pins the WZ→model mapping both Shadow Stars fixes rely
// on: the per-attack count (bulletCount) and the cast cost (bulletConsume) are
// distinct fields and must not be swapped.
func TestExtractBulletFields(t *testing.T) {
	m, err := Extract(RestModel{BulletCount: 3, BulletConsume: 200})
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if got := m.BulletCount(); got != 3 {
		t.Fatalf("BulletCount() = %d, want 3", got)
	}
	if got := m.BulletConsume(); got != 200 {
		t.Fatalf("BulletConsume() = %d, want 200", got)
	}
}
