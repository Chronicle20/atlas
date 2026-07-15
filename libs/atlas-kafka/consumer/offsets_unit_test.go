package consumer_test

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
)

// TestReplayableEnd pins the purged-partition collapse: a partition whose
// log-start offset has caught up to its high-water mark (retention deleted
// everything, or it was never written) holds nothing a FirstOffset consumer
// can replay, so its replayable end is 0 — the value catch-up gates treat as
// trivially caught up. Reporting the raw high-water mark instead wedges those
// gates forever (the 2026-07-08 world/channel/character-factory crash loops).
func TestReplayableEnd(t *testing.T) {
	cases := []struct {
		name        string
		first, last int64
		want        int64
	}{
		{"never written", 0, 0, 0},
		{"fully retained", 0, 5, 5},
		{"fully purged at 5", 5, 5, 0},
		{"fully purged at 2 (incident shape)", 2, 2, 0},
		{"partially purged", 2, 4, 4},
		{"single retained record", 4, 5, 5},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := consumer.ReplayableEnd(c.first, c.last); got != c.want {
				t.Fatalf("ReplayableEnd(%d, %d) = %d, want %d", c.first, c.last, got, c.want)
			}
		})
	}
}
