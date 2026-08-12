package chakra

import (
	"testing"
	"time"

	chakrastate "atlas-channel/character/chakra"

	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"

	channelhandler "atlas-channel/skill/handler"
)

// TestHealDelta pins the end-to-end heal: base = 2.9 x effective LUK,
// scaled by the window's snapshotted y, clamped to missing HP.
func TestHealDelta(t *testing.T) {
	tests := []struct {
		name  string
		y     int16
		luck  uint32
		hp    uint16
		maxHp uint16
		want  int16
	}{
		{"v83 L1 y=68", 68, 100, 100, 1000, 197},
		{"v83 L30 y=300", 300, 100, 100, 1000, 870},
		{"v48 L1 y=9", 9, 100, 100, 1000, 26},
		{"v48 L30 y=200", 200, 100, 100, 1000, 580},
		{"v95 L10 y=300", 300, 100, 100, 1000, 870},
		{"clamped to missing hp", 300, 100, 950, 1000, 50},
		{"at full hp", 300, 100, 1000, 1000, 0},
		{"zero luck", 300, 0, 100, 1000, 0},
		{"zero y", 0, 100, 100, 1000, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := chakrastate.Entry{SkillLevel: 1, X: 99, Y: tc.y, StartedAt: time.Now()}
			if got := healDelta(e, tc.luck, tc.hp, tc.maxHp); got != tc.want {
				t.Fatalf("healDelta = %d, want %d", got, tc.want)
			}
		})
	}
}

// TestRegisteredOnIdentity pins PRD FR-9.1: the handler is installed on the
// version-blind identity, so one registration covers all eleven provisioned
// versions without a raw wire-id comparison anywhere.
func TestRegisteredOnIdentity(t *testing.T) {
	if _, ok := channelhandler.Lookup(skill2.ChiefBanditChakra); !ok {
		t.Fatal("no Handler registered for skill2.ChiefBanditChakra")
	}
}
