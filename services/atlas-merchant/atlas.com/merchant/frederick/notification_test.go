package frederick

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNextTier(t *testing.T) {
	tests := []struct {
		name     string
		current  uint16
		expected uint16
		hasNext  bool
	}{
		{"zero advances to 2", 0, 2, true},
		{"2 advances to 5", 2, 5, true},
		{"5 advances to 10", 5, 10, true},
		{"10 advances to 15", 10, 15, true},
		{"15 advances to 30", 15, 30, true},
		{"30 advances to 60", 30, 60, true},
		{"60 advances to 90", 60, 90, true},
		{"90 has no next tier", 90, 0, false},
		{"between tiers 7 advances to 10", 7, 10, true},
		{"between tiers 3 advances to 5", 3, 5, true},
		{"between tiers 20 advances to 30", 20, 30, true},
		{"between tiers 45 advances to 60", 45, 60, true},
		{"above max 100 has no next tier", 100, 0, false},
		{"above max 200 has no next tier", 200, 0, false},
		{"1 advances to 2", 1, 2, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			next, hasNext := nextTier(tt.current)
			assert.Equal(t, tt.expected, next)
			assert.Equal(t, tt.hasNext, hasNext)
		})
	}
}
