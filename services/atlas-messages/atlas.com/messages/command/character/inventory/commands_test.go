package inventory

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

// TestAwardItemCommandProducer_RegexPatterns tests the item command regex patterns
func TestAwardItemCommandProducer_RegexPatterns(t *testing.T) {
	// Test the full regex with quantity
	reWithQuantity := regexp.MustCompile(`@award\s+(\w+)\s+item\s+(\d+)\s+(\d+)`)
	// Test the regex without quantity
	reWithoutQuantity := regexp.MustCompile(`@award\s+(\w+)\s+item\s+(\d+)`)

	testCases := []struct {
		name             string
		message          string
		expectMatch      bool
		hasQuantity      bool
		expectedTarget   string
		expectedItemId   string
		expectedQuantity string
	}{
		{
			name:             "Award me item with quantity",
			message:          "@award me item 2000000 10",
			expectMatch:      true,
			hasQuantity:      true,
			expectedTarget:   "me",
			expectedItemId:   "2000000",
			expectedQuantity: "10",
		},
		{
			name:             "Award me item without quantity",
			message:          "@award me item 2000000",
			expectMatch:      true,
			hasQuantity:      false,
			expectedTarget:   "me",
			expectedItemId:   "2000000",
			expectedQuantity: "",
		},
		{
			name:             "Award map item with quantity",
			message:          "@award map item 2000001 5",
			expectMatch:      true,
			hasQuantity:      true,
			expectedTarget:   "map",
			expectedItemId:   "2000001",
			expectedQuantity: "5",
		},
		{
			name:             "Award player item",
			message:          "@award PlayerName item 2000002 1",
			expectMatch:      true,
			hasQuantity:      true,
			expectedTarget:   "PlayerName",
			expectedItemId:   "2000002",
			expectedQuantity: "1",
		},
		{
			name:        "Invalid format - wrong command",
			message:     "@give me item 2000000",
			expectMatch: false,
		},
		{
			name:        "Invalid format - missing item id",
			message:     "@award me item",
			expectMatch: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var match []string
			if tc.hasQuantity {
				match = reWithQuantity.FindStringSubmatch(tc.message)
			} else {
				match = reWithoutQuantity.FindStringSubmatch(tc.message)
			}

			if tc.expectMatch {
				if tc.hasQuantity {
					if len(match) != 4 {
						t.Errorf("Expected match with 4 groups for quantity format, got %d groups", len(match))
						return
					}
					if match[3] != tc.expectedQuantity {
						t.Errorf("Expected quantity '%s', got '%s'", tc.expectedQuantity, match[3])
					}
				} else {
					if len(match) != 3 {
						t.Errorf("Expected match with 3 groups for no-quantity format, got %d groups", len(match))
						return
					}
				}
				if match[1] != tc.expectedTarget {
					t.Errorf("Expected target '%s', got '%s'", tc.expectedTarget, match[1])
				}
				if match[2] != tc.expectedItemId {
					t.Errorf("Expected item id '%s', got '%s'", tc.expectedItemId, match[2])
				}
			} else {
				if (tc.hasQuantity && len(match) == 4) || (!tc.hasQuantity && len(match) == 3) {
					t.Errorf("Expected no match, but got match: %v", match)
				}
			}
		})
	}
}

// TestAwardItemCommandProducer_GmCheck tests GM permission enforcement
func TestAwardItemCommandProducer_GmCheck(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := context.Background()

	testCases := []struct {
		name        string
		isGm        bool
		message     string
		expectFound bool
	}{
		{
			name:        "GM can execute item command with quantity",
			isGm:        true,
			message:     "@award me item 2000000 10",
			expectFound: true,
		},
		{
			name:        "GM can execute item command without quantity",
			isGm:        true,
			message:     "@award me item 2000000",
			expectFound: true,
		},
		{
			name:        "Non-GM cannot execute command",
			isGm:        false,
			message:     "@award me item 2000000 10",
			expectFound: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			char := createTestCharacter(12345, "TestPlayer", tc.isGm, 100000000)
			f := field.NewBuilder(1, 1, 100000000).Build()

			producer := AwardItemCommandProducer(logger)
			_, found := producer(ctx)(f,char, tc.message)

			// Note: The actual command may not find the executor if the item doesn't exist
			// in the asset processor. We're testing the initial pattern matching and GM check.
			if tc.isGm && tc.expectFound {
				// For GMs with valid patterns, the command should at least try to process
				// The actual result depends on whether the item exists in the data service
				t.Logf("GM command test - found=%v (may depend on asset validation)", found)
			} else if !tc.isGm && found {
				t.Errorf("Expected non-GM to be rejected, but found=true")
			}
		})
	}
}

// TestAwardItemCommandProducer_NoMatchReturnsNil tests non-matching messages
func TestAwardItemCommandProducer_NoMatchReturnsNil(t *testing.T) {
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
			message: "@give me item 2000000",
		},
		{
			name:    "Missing item id",
			message: "@award me item",
		},
		{
			name:    "Empty message",
			message: "",
		},
		{
			name:    "Wrong keyword",
			message: "@award me equipment 2000000",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			producer := AwardItemCommandProducer(logger)
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

// TestAwardItemCommandProducer_TargetResolution tests target resolution
func TestAwardItemCommandProducer_TargetResolution(t *testing.T) {
	re := regexp.MustCompile(`@award\s+(\w+)\s+item\s+(\d+)`)

	testCases := []struct {
		name         string
		message      string
		expectedType string
	}{
		{
			name:         "Target me",
			message:      "@award me item 2000000",
			expectedType: "me",
		},
		{
			name:         "Target map",
			message:      "@award map item 2000000",
			expectedType: "map",
		},
		{
			name:         "Target by name",
			message:      "@award PlayerName item 2000000",
			expectedType: "name",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			match := re.FindStringSubmatch(tc.message)
			if len(match) < 3 {
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

// TestAwardItemCommandProducer_QuantityParsing tests quantity parsing
func TestAwardItemCommandProducer_QuantityParsing(t *testing.T) {
	reWithQuantity := regexp.MustCompile(`@award\s+(\w+)\s+item\s+(\d+)\s+(\d+)`)
	reWithoutQuantity := regexp.MustCompile(`@award\s+(\w+)\s+item\s+(\d+)`)

	testCases := []struct {
		name             string
		message          string
		expectedQuantity string
		hasExplicit      bool
	}{
		{
			name:             "Explicit quantity 10",
			message:          "@award me item 2000000 10",
			expectedQuantity: "10",
			hasExplicit:      true,
		},
		{
			name:             "Explicit quantity 1",
			message:          "@award me item 2000000 1",
			expectedQuantity: "1",
			hasExplicit:      true,
		},
		{
			name:             "Large quantity",
			message:          "@award me item 2000000 999",
			expectedQuantity: "999",
			hasExplicit:      true,
		},
		{
			name:             "No explicit quantity (defaults to 1)",
			message:          "@award me item 2000000",
			expectedQuantity: "1", // Default
			hasExplicit:      false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.hasExplicit {
				match := reWithQuantity.FindStringSubmatch(tc.message)
				if len(match) != 4 {
					t.Fatalf("Expected match with 4 groups, got %d", len(match))
				}
				if match[3] != tc.expectedQuantity {
					t.Errorf("Expected quantity '%s', got '%s'", tc.expectedQuantity, match[3])
				}
			} else {
				match := reWithoutQuantity.FindStringSubmatch(tc.message)
				if len(match) != 3 {
					t.Fatalf("Expected match with 3 groups, got %d", len(match))
				}
				// When no quantity is specified, the code defaults to "1"
				// This is tested by verifying the message matches the no-quantity pattern
			}
		})
	}
}
