package pet

import (
	"atlas-messages/character"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/sirupsen/logrus/hooks/test"
	"golang.org/x/net/context"
)

// createTestCharacter creates a character model for testing.
func createTestCharacter(id uint32, name string, isGm bool) character.Model {
	gm := 0
	if isGm {
		gm = 1
	}
	return character.NewModelBuilder().
		SetId(id).
		SetName(name).
		SetGm(gm).
		SetAccountId(100).
		Build()
}

// TestAwardTamenessCommand_MatchesAndGmGated verifies that a GM issuing
// "@award me tameness 2000" produces an executor, a non-GM is rejected, and a
// non-matching meso command is not handled by this producer.
func TestAwardTamenessCommand_MatchesAndGmGated(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := context.Background()
	f := field.NewBuilder(1, 1, 100000000).Build()

	testCases := []struct {
		name        string
		isGm        bool
		message     string
		expectFound bool
	}{
		{
			name:        "GM tameness command",
			isGm:        true,
			message:     "@award me tameness 2000",
			expectFound: true,
		},
		{
			name:        "Non-GM tameness command",
			isGm:        false,
			message:     "@award me tameness 2000",
			expectFound: false,
		},
		{
			name:        "GM meso command does not match",
			isGm:        true,
			message:     "@award me meso 5",
			expectFound: false,
		},
		{
			name:        "GM tameness for a multi-word pet name",
			isGm:        true,
			message:     "@award Baby Dragon tameness 2000",
			expectFound: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			char := createTestCharacter(12345, "TestPlayer", tc.isGm)

			producer := AwardTamenessCommandProducer(logger)
			executor, found := producer(ctx)(f, char, tc.message)

			if found != tc.expectFound {
				t.Errorf("Expected found=%v for message %q (gm=%v), got found=%v", tc.expectFound, tc.message, tc.isGm, found)
			}
			if !tc.expectFound && executor != nil {
				t.Errorf("Expected nil executor for non-matching/ungated message %q", tc.message)
			}
		})
	}
}
