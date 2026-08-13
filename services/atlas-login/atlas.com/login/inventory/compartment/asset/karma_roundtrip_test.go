package asset

import (
	"testing"

	"github.com/google/uuid"

	af "github.com/Chronicle20/atlas/libs/atlas-constants/asset"
)

// TestKarmaUsedRoundTrip is the FR-4.4 regression guard. Before task-223,
// SetKarmaUsed wrote FlagKarmaEquip (0x10) while KarmaUsed read FlagKarmaUse
// (0x02), so a set NEVER read back for any asset.
func TestKarmaUsedRoundTrip(t *testing.T) {
	testCases := []struct {
		name       string
		templateId uint32
	}{
		{"equip", 1002357},
		{"bundle", 2280000},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := NewBuilder(uuid.New(), tc.templateId).SetKarmaUsed(true).Build()
			if !m.KarmaUsed() {
				t.Fatalf("KarmaUsed() = false after SetKarmaUsed(true) for template %d", tc.templateId)
			}
			cleared := Clone(m).SetKarmaUsed(false).Build()
			if cleared.KarmaUsed() {
				t.Fatalf("KarmaUsed() = true after SetKarmaUsed(false) for template %d", tc.templateId)
			}
		})
	}
}

// TestKarmaUsedLeavesSpikesAlone is the FR-4.5 guard: 0x02 is FlagSpikes on an
// EQUIP, so a karma mark on an equip must not render spikes, and clearing karma
// on an equip must not silently clear a genuine spikes flag.
func TestKarmaUsedLeavesSpikesAlone(t *testing.T) {
	spiked := NewBuilder(uuid.New(), 1002357).AddFlag(af.FlagSpikes).SetKarmaUsed(true).Build()
	if !spiked.Spikes() {
		t.Fatal("Spikes() = false after karma-marking a spiked equip")
	}
	if !spiked.KarmaUsed() {
		t.Fatal("KarmaUsed() = false on a spiked equip")
	}

	plain := NewBuilder(uuid.New(), 1002357).SetKarmaUsed(true).Build()
	if plain.Spikes() {
		t.Fatal("Spikes() = true after karma-marking an unspiked equip; the wrong bit was written")
	}
}
