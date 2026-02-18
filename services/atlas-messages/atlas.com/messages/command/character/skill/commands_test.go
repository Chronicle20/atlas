package skill

import (
	"atlas-messages/character"
	"regexp"
	"testing"

	"github.com/Chronicle20/atlas-constants/field"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/sirupsen/logrus/hooks/test"
	"golang.org/x/net/context"
)

// createTestCharacter creates a character model for testing
func createTestCharacter(id uint32, name string, isGm bool, mapId _map.Id) character.Model {
	gm := 0
	if isGm {
		gm = 1
	}
	return character.NewModelBuilder().
		SetId(id).
		SetName(name).
		SetGm(gm).
		SetMapId(mapId).
		SetAccountId(100).
		Build()
}

// TestMaxSkillCommandProducer_RegexPatterns tests the max skill command regex
func TestMaxSkillCommandProducer_RegexPatterns(t *testing.T) {
	re := regexp.MustCompile(`@skill\s+max\s+(\d+)`)

	testCases := []struct {
		name            string
		message         string
		expectMatch     bool
		expectedSkillId string
	}{
		{
			name:            "Valid max skill command",
			message:         "@skill max 1001",
			expectMatch:     true,
			expectedSkillId: "1001",
		},
		{
			name:            "Max skill with large id",
			message:         "@skill max 3121002",
			expectMatch:     true,
			expectedSkillId: "3121002",
		},
		{
			name:        "Invalid format - missing skill id",
			message:     "@skill max",
			expectMatch: false,
		},
		{
			name:        "Invalid format - wrong command",
			message:     "@skills max 1001",
			expectMatch: false,
		},
		{
			name:        "Invalid format - wrong order",
			message:     "@skill 1001 max",
			expectMatch: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			match := re.FindStringSubmatch(tc.message)

			if tc.expectMatch {
				if len(match) != 2 {
					t.Errorf("Expected match with 2 groups, got %d groups", len(match))
					return
				}
				if match[1] != tc.expectedSkillId {
					t.Errorf("Expected skill id '%s', got '%s'", tc.expectedSkillId, match[1])
				}
			} else {
				if len(match) == 2 {
					t.Errorf("Expected no match, but got match: %v", match)
				}
			}
		})
	}
}

// TestResetSkillCommandProducer_RegexPatterns tests the reset skill command regex
func TestResetSkillCommandProducer_RegexPatterns(t *testing.T) {
	re := regexp.MustCompile(`@skill\s+reset\s+(\d+)`)

	testCases := []struct {
		name            string
		message         string
		expectMatch     bool
		expectedSkillId string
	}{
		{
			name:            "Valid reset skill command",
			message:         "@skill reset 1001",
			expectMatch:     true,
			expectedSkillId: "1001",
		},
		{
			name:            "Reset skill with large id",
			message:         "@skill reset 3121002",
			expectMatch:     true,
			expectedSkillId: "3121002",
		},
		{
			name:        "Invalid format - missing skill id",
			message:     "@skill reset",
			expectMatch: false,
		},
		{
			name:        "Invalid format - wrong command",
			message:     "@skills reset 1001",
			expectMatch: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			match := re.FindStringSubmatch(tc.message)

			if tc.expectMatch {
				if len(match) != 2 {
					t.Errorf("Expected match with 2 groups, got %d groups", len(match))
					return
				}
				if match[1] != tc.expectedSkillId {
					t.Errorf("Expected skill id '%s', got '%s'", tc.expectedSkillId, match[1])
				}
			} else {
				if len(match) == 2 {
					t.Errorf("Expected no match, but got match: %v", match)
				}
			}
		})
	}
}

// TestMaxSkillCommandProducer_NoMatchReturnsNil tests non-matching messages
func TestMaxSkillCommandProducer_NoMatchReturnsNil(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := context.Background()
	gmChar := createTestCharacter(12345, "TestGM", true, 100000000)
	f := field.NewBuilder(1, 1, 100000000).Build()

	testCases := []struct {
		name    string
		message string
	}{
		{
			name:    "Wrong command",
			message: "@skills max 1001",
		},
		{
			name:    "Missing skill id",
			message: "@skill max",
		},
		{
			name:    "Empty message",
			message: "",
		},
		{
			name:    "Wrong operation",
			message: "@skill add 1001",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			producer := MaxSkillCommandProducer(logger)
			executor, found := producer(ctx)(f,gmChar, tc.message)

			if found {
				t.Errorf("Expected found=false for message '%s', got found=true", tc.message)
			}

			if executor != nil {
				t.Errorf("Expected executor to be nil for non-matching message")
			}
		})
	}
}

// TestResetSkillCommandProducer_NoMatchReturnsNil tests non-matching messages
func TestResetSkillCommandProducer_NoMatchReturnsNil(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := context.Background()
	gmChar := createTestCharacter(12345, "TestGM", true, 100000000)
	f := field.NewBuilder(1, 1, 100000000).Build()

	testCases := []struct {
		name    string
		message string
	}{
		{
			name:    "Wrong command",
			message: "@skills reset 1001",
		},
		{
			name:    "Missing skill id",
			message: "@skill reset",
		},
		{
			name:    "Empty message",
			message: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			producer := ResetSkillCommandProducer(logger)
			executor, found := producer(ctx)(f,gmChar, tc.message)

			if found {
				t.Errorf("Expected found=false for message '%s', got found=true", tc.message)
			}

			if executor != nil {
				t.Errorf("Expected executor to be nil for non-matching message")
			}
		})
	}
}

// TestSkillCommandProducers_SkillIdParsing tests skill ID parsing
func TestSkillCommandProducers_SkillIdParsing(t *testing.T) {
	re := regexp.MustCompile(`@skill\s+(max|reset)\s+(\d+)`)

	testCases := []struct {
		name            string
		message         string
		expectedOp      string
		expectedSkillId string
	}{
		{
			name:            "Max skill 1001",
			message:         "@skill max 1001",
			expectedOp:      "max",
			expectedSkillId: "1001",
		},
		{
			name:            "Reset skill 2001",
			message:         "@skill reset 2001",
			expectedOp:      "reset",
			expectedSkillId: "2001",
		},
		{
			name:            "Warrior skill",
			message:         "@skill max 1121002",
			expectedOp:      "max",
			expectedSkillId: "1121002",
		},
		{
			name:            "Mage skill",
			message:         "@skill max 2121002",
			expectedOp:      "max",
			expectedSkillId: "2121002",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			match := re.FindStringSubmatch(tc.message)
			if len(match) != 3 {
				t.Fatalf("Expected match with 3 groups, got %d", len(match))
			}

			if match[1] != tc.expectedOp {
				t.Errorf("Expected operation '%s', got '%s'", tc.expectedOp, match[1])
			}

			if match[2] != tc.expectedSkillId {
				t.Errorf("Expected skill id '%s', got '%s'", tc.expectedSkillId, match[2])
			}
		})
	}
}
