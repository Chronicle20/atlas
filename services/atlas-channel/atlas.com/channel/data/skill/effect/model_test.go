package effect

import (
	"testing"
)

func TestRecoveryAccessors(t *testing.T) {
	m, err := Extract(RestModel{Hp: 100, Mp: 50, HPR: 0.5, MPR: 0.25})
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}
	if got := m.HP(); got != 100 {
		t.Errorf("HP() = %d, want 100", got)
	}
	if got := m.MP(); got != 50 {
		t.Errorf("MP() = %d, want 50", got)
	}
	if got := m.HpR(); got != 0.5 {
		t.Errorf("HpR() = %v, want 0.5", got)
	}
	if got := m.MpR(); got != 0.25 {
		t.Errorf("MpR() = %v, want 0.25", got)
	}
}
