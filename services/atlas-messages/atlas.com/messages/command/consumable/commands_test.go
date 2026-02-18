package consumable

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

// TestConsumeCommandProducer_RegexPatterns tests the regex pattern matching
func TestConsumeCommandProducer_RegexPatterns(t *testing.T) {
	re := regexp.MustCompile(`^@consume\s+(\w+)\s+(\d+)$`)

	testCases := []struct {
		name          string
		message       string
		expectMatch   bool
		expectedName  string
		expectedValue string
	}{
		{
			name:          "Use item on me",
			message:       "@consume me 2022245",
			expectMatch:   true,
			expectedName:  "me",
			expectedValue: "2022245",
		},
		{
			name:          "Use item on map",
			message:       "@consume map 2022245",
			expectMatch:   true,
			expectedName:  "map",
			expectedValue: "2022245",
		},
		{
			name:          "Use item on player by name",
			message:       "@consume PlayerName 2022245",
			expectMatch:   true,
			expectedName:  "PlayerName",
			expectedValue: "2022245",
		},
		{
			name:        "Invalid format - missing item id",
			message:     "@consume me",
			expectMatch: false,
		},
		{
			name:        "Invalid format - wrong command",
			message:     "@applyitem me 2022245",
			expectMatch: false,
		},
		{
			name:        "Empty message",
			message:     "",
			expectMatch: false,
		},
		{
			name:        "Invalid format - extra arguments",
			message:     "@consume me 2022245 extra",
			expectMatch: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			match := re.FindStringSubmatch(tc.message)

			if tc.expectMatch {
				if len(match) != 3 {
					t.Errorf("Expected match with 3 groups, got %d groups", len(match))
					return
				}
				if match[1] != tc.expectedName {
					t.Errorf("Expected target '%s', got '%s'", tc.expectedName, match[1])
				}
				if match[2] != tc.expectedValue {
					t.Errorf("Expected item id '%s', got '%s'", tc.expectedValue, match[2])
				}
			} else {
				if len(match) == 3 {
					t.Errorf("Expected no match, but got match: %v", match)
				}
			}
		})
	}
}

// TestConsumeCommandProducer_GmCheck tests GM permission enforcement
func TestConsumeCommandProducer_GmCheck(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := context.Background()

	testCases := []struct {
		name        string
		isGm        bool
		message     string
		expectFound bool
	}{
		{
			name:        "GM can execute command",
			isGm:        true,
			message:     "@consume me 2022245",
			expectFound: true,
		},
		{
			name:        "Non-GM cannot execute command",
			isGm:        false,
			message:     "@consume me 2022245",
			expectFound: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			char := createTestCharacter(12345, "TestPlayer", tc.isGm, 100000000)

			producer := ConsumeCommandProducer(logger)
			f := field.NewBuilder(1, 1, 100000000).Build()
			_, found := producer(ctx)(f, char, tc.message)

			if found != tc.expectFound {
				t.Errorf("Expected found=%v for GM=%v, got found=%v", tc.expectFound, tc.isGm, found)
			}
		})
	}
}

// TestConsumeCommandProducer_NoMatchReturnsNil tests that non-matching commands return nil executor
func TestConsumeCommandProducer_NoMatchReturnsNil(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := context.Background()
	gmChar := createTestCharacter(12345, "TestGM", true, 100000000)

	testCases := []struct {
		name    string
		message string
	}{
		{
			name:    "Wrong command name",
			message: "@applyitem me 2022245",
		},
		{
			name:    "Missing item id",
			message: "@consume me",
		},
		{
			name:    "Empty message",
			message: "",
		},
		{
			name:    "Non-numeric item id",
			message: "@consume me potion",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			f := field.NewBuilder(1, 1, 100000000).Build()
			executor, found := ConsumeCommandProducer(logger)(ctx)(f, gmChar, tc.message)

			if found {
				t.Errorf("Expected found=false for message '%s', got found=true", tc.message)
			}

			if executor != nil {
				t.Errorf("Expected executor to be nil for non-matching message")
			}
		})
	}
}

// TestTargetResolution tests the different target types (me, map, name)
func TestTargetResolution(t *testing.T) {
	testCases := []struct {
		name         string
		message      string
		expectedType string
	}{
		{
			name:         "Target me",
			message:      "@consume me 2022245",
			expectedType: "me",
		},
		{
			name:         "Target map",
			message:      "@consume map 2022245",
			expectedType: "map",
		},
		{
			name:         "Target by name",
			message:      "@consume PlayerName 2022245",
			expectedType: "name",
		},
	}

	re := regexp.MustCompile(`^@consume\s+(\w+)\s+(\d+)$`)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			match := re.FindStringSubmatch(tc.message)
			if len(match) != 3 {
				t.Fatal("Expected regex to match")
			}

			target := match[1]
			var actualType string
			switch target {
			case "me":
				actualType = "me"
			case "map":
				actualType = "map"
			default:
				actualType = "name"
			}

			if actualType != tc.expectedType {
				t.Errorf("Expected target type '%s', got '%s' for target '%s'", tc.expectedType, actualType, target)
			}
		})
	}
}
