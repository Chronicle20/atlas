package _map

import (
	"atlas-messages/character"
	"regexp"
	"testing"

	"github.com/sirupsen/logrus/hooks/test"
	"golang.org/x/net/context"
)

// createTestCharacter creates a character model for testing
func createTestCharacter(id uint32, name string, isGm bool, mapId uint32) character.Model {
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

// TestWarpCommandProducer_RegexPatterns tests the warp command regex
func TestWarpCommandProducer_RegexPatterns(t *testing.T) {
	re := regexp.MustCompile(`@warp\s+(\w+)\s+(\d+)`)

	testCases := []struct {
		name           string
		message        string
		expectMatch    bool
		expectedTarget string
		expectedMapId  string
	}{
		{
			name:           "Warp me to map",
			message:        "@warp me 100000000",
			expectMatch:    true,
			expectedTarget: "me",
			expectedMapId:  "100000000",
		},
		{
			name:           "Warp map (all in map)",
			message:        "@warp map 200000000",
			expectMatch:    true,
			expectedTarget: "map",
			expectedMapId:  "200000000",
		},
		{
			name:           "Warp player by name",
			message:        "@warp PlayerName 100000001",
			expectMatch:    true,
			expectedTarget: "PlayerName",
			expectedMapId:  "100000001",
		},
		{
			name:        "Invalid format - missing map id",
			message:     "@warp me",
			expectMatch: false,
		},
		{
			name:        "Invalid format - wrong command",
			message:     "@teleport me 100000000",
			expectMatch: false,
		},
		{
			name:        "Invalid format - no target",
			message:     "@warp 100000000",
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
				if match[1] != tc.expectedTarget {
					t.Errorf("Expected target '%s', got '%s'", tc.expectedTarget, match[1])
				}
				if match[2] != tc.expectedMapId {
					t.Errorf("Expected map id '%s', got '%s'", tc.expectedMapId, match[2])
				}
			} else {
				if len(match) == 3 {
					t.Errorf("Expected no match, but got match: %v", match)
				}
			}
		})
	}
}

// TestWhereAmICommandProducer_RegexPatterns tests the query map command regex
func TestWhereAmICommandProducer_RegexPatterns(t *testing.T) {
	re := regexp.MustCompile(`@query map`)

	testCases := []struct {
		name        string
		message     string
		expectMatch bool
	}{
		{
			name:        "Valid query map command",
			message:     "@query map",
			expectMatch: true,
		},
		{
			name:        "Invalid format - wrong command",
			message:     "@query location",
			expectMatch: false,
		},
		{
			name:        "Invalid format - different command",
			message:     "@where am i",
			expectMatch: false,
		},
		{
			name:        "Extra text after command",
			message:     "@query map 123",
			expectMatch: true, // Regex matches the beginning
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			match := re.FindStringSubmatch(tc.message)

			if tc.expectMatch {
				if len(match) != 1 {
					t.Errorf("Expected match with 1 group, got %d groups", len(match))
				}
			} else {
				if len(match) == 1 {
					t.Errorf("Expected no match, but got match: %v", match)
				}
			}
		})
	}
}

// TestWarpCommandProducer_GmCheck tests GM permission enforcement
func TestWarpCommandProducer_GmCheck(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := context.Background()

	testCases := []struct {
		name        string
		isGm        bool
		message     string
		expectFound bool
	}{
		{
			name:        "GM can execute warp command",
			isGm:        true,
			message:     "@warp me 100000000",
			expectFound: true,
		},
		{
			name:        "Non-GM cannot execute warp command",
			isGm:        false,
			message:     "@warp me 100000000",
			expectFound: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			char := createTestCharacter(12345, "TestPlayer", tc.isGm, 100000000)

			producer := WarpCommandProducer(logger)
			_, found := producer(ctx)(1, 1, char, tc.message)

			if found != tc.expectFound {
				t.Errorf("Expected found=%v for GM=%v, got found=%v", tc.expectFound, tc.isGm, found)
			}
		})
	}
}

// TestWhereAmICommandProducer_NoGmCheck tests that query map doesn't require GM
func TestWhereAmICommandProducer_NoGmCheck(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := context.Background()

	testCases := []struct {
		name        string
		isGm        bool
		message     string
		expectFound bool
	}{
		{
			name:        "GM can execute query map",
			isGm:        true,
			message:     "@query map",
			expectFound: true,
		},
		{
			name:        "Non-GM can also execute query map",
			isGm:        false,
			message:     "@query map",
			expectFound: true, // This command doesn't require GM
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			char := createTestCharacter(12345, "TestPlayer", tc.isGm, 100000000)

			producer := WhereAmICommandProducer(logger)
			_, found := producer(ctx)(1, 1, char, tc.message)

			if found != tc.expectFound {
				t.Errorf("Expected found=%v for GM=%v, got found=%v", tc.expectFound, tc.isGm, found)
			}
		})
	}
}

// TestWarpCommandProducer_NoMatchReturnsNil tests non-matching messages
func TestWarpCommandProducer_NoMatchReturnsNil(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := context.Background()
	gmChar := createTestCharacter(12345, "TestGM", true, 100000000)

	testCases := []struct {
		name    string
		message string
	}{
		{
			name:    "Wrong command",
			message: "@teleport me 100000000",
		},
		{
			name:    "Missing map id",
			message: "@warp me",
		},
		{
			name:    "Empty message",
			message: "",
		},
		{
			name:    "Missing target",
			message: "@warp 100000000",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			producer := WarpCommandProducer(logger)
			executor, found := producer(ctx)(1, 1, gmChar, tc.message)

			if found {
				t.Errorf("Expected found=false for message '%s', got found=true", tc.message)
			}

			if executor != nil {
				t.Errorf("Expected executor to be nil for non-matching message")
			}
		})
	}
}

// TestWarpCommandProducer_TargetResolution tests target resolution
func TestWarpCommandProducer_TargetResolution(t *testing.T) {
	re := regexp.MustCompile(`@warp\s+(\w+)\s+(\d+)`)

	testCases := []struct {
		name         string
		message      string
		expectedType string
	}{
		{
			name:         "Target me",
			message:      "@warp me 100000000",
			expectedType: "me",
		},
		{
			name:         "Target map (all in map)",
			message:      "@warp map 100000000",
			expectedType: "map",
		},
		{
			name:         "Target by name",
			message:      "@warp PlayerName 100000000",
			expectedType: "name",
		},
	}

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

// TestWarpCommandProducer_MapIdParsing tests map ID parsing
func TestWarpCommandProducer_MapIdParsing(t *testing.T) {
	re := regexp.MustCompile(`@warp\s+(\w+)\s+(\d+)`)

	testCases := []struct {
		name          string
		message       string
		expectedMapId string
	}{
		{
			name:          "Henesys",
			message:       "@warp me 100000000",
			expectedMapId: "100000000",
		},
		{
			name:          "Ellinia",
			message:       "@warp me 101000000",
			expectedMapId: "101000000",
		},
		{
			name:          "Orbis",
			message:       "@warp me 200000000",
			expectedMapId: "200000000",
		},
		{
			name:          "Small map id",
			message:       "@warp me 1",
			expectedMapId: "1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			match := re.FindStringSubmatch(tc.message)
			if len(match) != 3 {
				t.Fatalf("Expected match with 3 groups, got %d", len(match))
			}

			if match[2] != tc.expectedMapId {
				t.Errorf("Expected map id '%s', got '%s'", tc.expectedMapId, match[2])
			}
		})
	}
}
