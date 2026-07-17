package effect

import "testing"

func TestModelBulletCount(t *testing.T) {
	m := Model{bulletCount: 200}
	if got := m.BulletCount(); got != 200 {
		t.Fatalf("BulletCount() = %d, want 200", got)
	}
}
