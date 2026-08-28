package _map

import (
	"testing"

	"github.com/sirupsen/logrus/hooks/test"
	"golang.org/x/net/context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

// TestBackEffectCommandProducer_Matching tests regex matching and GM gating
// for the @backeffect command.
func TestBackEffectCommandProducer_Matching(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := context.Background()

	testCases := []struct {
		name        string
		isGm        bool
		message     string
		expectFound bool
	}{
		{
			name:        "set with duration",
			isGm:        true,
			message:     "@backeffect 1 0 1000",
			expectFound: true,
		},
		{
			name:        "set without duration",
			isGm:        true,
			message:     "@backeffect 1 0",
			expectFound: true,
		},
		{
			name:        "hide",
			isGm:        true,
			message:     "@backeffect 2 1",
			expectFound: true,
		},
		{
			name:        "non-gm rejected",
			isGm:        false,
			message:     "@backeffect 1 0",
			expectFound: false,
		},
		{
			name:        "effect out of range",
			isGm:        true,
			message:     "@backeffect 1 2",
			expectFound: false,
		},
		{
			name:        "missing effect",
			isGm:        true,
			message:     "@backeffect 1",
			expectFound: false,
		},
		{
			name:        "unrelated message",
			isGm:        true,
			message:     "@weather 5120016 hi",
			expectFound: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			char, mapId := createTestCharacter(t, 12345, "TestGM", tc.isGm, 100000000)
			f := field.NewBuilder(1, 1, mapId).Build()

			producer := BackEffectCommandProducer(logger)
			_, found := producer(ctx)(f, char, tc.message)

			if found != tc.expectFound {
				t.Errorf("Expected found=%v for message '%s' (gm=%v), got found=%v", tc.expectFound, tc.message, tc.isGm, found)
			}
		})
	}
}

// TestClearBackEffectCommandProducer_Matching tests regex matching and GM
// gating for the @clearbackeffect command.
func TestClearBackEffectCommandProducer_Matching(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := context.Background()

	testCases := []struct {
		name        string
		isGm        bool
		message     string
		expectFound bool
	}{
		{
			name:        "gm clear",
			isGm:        true,
			message:     "@clearbackeffect",
			expectFound: true,
		},
		{
			name:        "non-gm rejected",
			isGm:        false,
			message:     "@clearbackeffect",
			expectFound: false,
		},
		{
			name:        "unrelated message",
			isGm:        true,
			message:     "@backeffect 1 0",
			expectFound: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			char, mapId := createTestCharacter(t, 12345, "TestGM", tc.isGm, 100000000)
			f := field.NewBuilder(1, 1, mapId).Build()

			producer := ClearBackEffectCommandProducer(logger)
			_, found := producer(ctx)(f, char, tc.message)

			if found != tc.expectFound {
				t.Errorf("Expected found=%v for message '%s' (gm=%v), got found=%v", tc.expectFound, tc.message, tc.isGm, found)
			}
		})
	}
}
