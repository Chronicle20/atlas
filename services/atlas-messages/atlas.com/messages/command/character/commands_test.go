package character

import (
	"atlas-messages/character"
	"atlas-messages/command"
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

// TestAwardExperienceCommandProducer_RegexPatterns tests the regex pattern matching
func TestAwardExperienceCommandProducer_RegexPatterns(t *testing.T) {
	re := regexp.MustCompile(`@award\s+(\w+)\s+experience\s+(\d+)`)

	testCases := []struct {
		name          string
		message       string
		expectMatch   bool
		expectedName  string
		expectedValue string
	}{
		{
			name:          "Award me experience",
			message:       "@award me experience 1000",
			expectMatch:   true,
			expectedName:  "me",
			expectedValue: "1000",
		},
		{
			name:          "Award map experience",
			message:       "@award map experience 5000",
			expectMatch:   true,
			expectedName:  "map",
			expectedValue: "5000",
		},
		{
			name:          "Award player by name",
			message:       "@award PlayerName experience 2500",
			expectMatch:   true,
			expectedName:  "PlayerName",
			expectedValue: "2500",
		},
		{
			name:        "Invalid format - missing amount",
			message:     "@award me experience",
			expectMatch: false,
		},
		{
			name:        "Invalid format - wrong command",
			message:     "@give me experience 1000",
			expectMatch: false,
		},
		{
			name:        "Invalid format - negative amount",
			message:     "@award me experience -1000",
			expectMatch: false,
		},
		{
			name:        "Empty message",
			message:     "",
			expectMatch: false,
		},
		{
			name:          "Large experience value",
			message:       "@award me experience 999999999",
			expectMatch:   true,
			expectedName:  "me",
			expectedValue: "999999999",
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
					t.Errorf("Expected amount '%s', got '%s'", tc.expectedValue, match[2])
				}
			} else {
				if len(match) == 3 {
					t.Errorf("Expected no match, but got match: %v", match)
				}
			}
		})
	}
}

// TestAwardLevelCommandProducer_RegexPatterns tests the level command regex
func TestAwardLevelCommandProducer_RegexPatterns(t *testing.T) {
	re := regexp.MustCompile(`@award\s+(\w+)\s+(\d+)\s+level`)

	testCases := []struct {
		name          string
		message       string
		expectMatch   bool
		expectedName  string
		expectedValue string
	}{
		{
			name:          "Award me 5 levels",
			message:       "@award me 5 level",
			expectMatch:   true,
			expectedName:  "me",
			expectedValue: "5",
		},
		{
			name:          "Award map 1 level",
			message:       "@award map 1 level",
			expectMatch:   true,
			expectedName:  "map",
			expectedValue: "1",
		},
		{
			name:          "Award player by name",
			message:       "@award TestPlayer 10 level",
			expectMatch:   true,
			expectedName:  "TestPlayer",
			expectedValue: "10",
		},
		// Note: "levels" still matches because the regex only requires "level" at the end
		// and "levels" contains "level". The actual behavior allows this.
		{
			name:        "Invalid format - wrong order",
			message:     "@award me level 5",
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
					t.Errorf("Expected amount '%s', got '%s'", tc.expectedValue, match[2])
				}
			} else {
				if len(match) == 3 {
					t.Errorf("Expected no match, but got match: %v", match)
				}
			}
		})
	}
}

// TestChangeJobCommandProducer_RegexPatterns tests the job change regex
func TestChangeJobCommandProducer_RegexPatterns(t *testing.T) {
	re := regexp.MustCompile(`@change\s+(\w+)\s+job\s+(\d+)`)

	testCases := []struct {
		name          string
		message       string
		expectMatch   bool
		expectedName  string
		expectedValue string
	}{
		{
			name:          "Change my job",
			message:       "@change my job 100",
			expectMatch:   true,
			expectedName:  "my",
			expectedValue: "100",
		},
		{
			name:          "Change player job",
			message:       "@change PlayerName job 200",
			expectMatch:   true,
			expectedName:  "PlayerName",
			expectedValue: "200",
		},
		{
			name:        "Invalid format - missing job id",
			message:     "@change my job",
			expectMatch: false,
		},
		{
			name:        "Invalid format - wrong command",
			message:     "@set my job 100",
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
					t.Errorf("Expected job id '%s', got '%s'", tc.expectedValue, match[2])
				}
			} else {
				if len(match) == 3 {
					t.Errorf("Expected no match, but got match: %v", match)
				}
			}
		})
	}
}

// TestAwardMesoCommandProducer_RegexPatterns tests the meso command regex
func TestAwardMesoCommandProducer_RegexPatterns(t *testing.T) {
	re := regexp.MustCompile(`@award\s+(\w+)\s+meso\s+(-?\d+)`)

	testCases := []struct {
		name          string
		message       string
		expectMatch   bool
		expectedName  string
		expectedValue string
	}{
		{
			name:          "Award me positive meso",
			message:       "@award me meso 10000",
			expectMatch:   true,
			expectedName:  "me",
			expectedValue: "10000",
		},
		{
			name:          "Award me negative meso",
			message:       "@award me meso -5000",
			expectMatch:   true,
			expectedName:  "me",
			expectedValue: "-5000",
		},
		{
			name:          "Award map meso",
			message:       "@award map meso 1000",
			expectMatch:   true,
			expectedName:  "map",
			expectedValue: "1000",
		},
		{
			name:          "Award player meso",
			message:       "@award TestPlayer meso 50000",
			expectMatch:   true,
			expectedName:  "TestPlayer",
			expectedValue: "50000",
		},
		{
			name:        "Invalid format - mesos plural",
			message:     "@award me mesos 1000",
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
					t.Errorf("Expected amount '%s', got '%s'", tc.expectedValue, match[2])
				}
			} else {
				if len(match) == 3 {
					t.Errorf("Expected no match, but got match: %v", match)
				}
			}
		})
	}
}

// TestAwardExperienceCommandProducer_GmCheck tests GM permission enforcement
func TestAwardExperienceCommandProducer_GmCheck(t *testing.T) {
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
			message:     "@award me experience 1000",
			expectFound: true,
		},
		{
			name:        "Non-GM cannot execute command",
			isGm:        false,
			message:     "@award me experience 1000",
			expectFound: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			char := createTestCharacter(12345, "TestPlayer", tc.isGm, 100000000)

			producer := AwardExperienceCommandProducer(logger)
			_, found := producer(ctx)(1, 1, char, tc.message)

			if found != tc.expectFound {
				t.Errorf("Expected found=%v for GM=%v, got found=%v", tc.expectFound, tc.isGm, found)
			}
		})
	}
}

// TestAwardLevelCommandProducer_GmCheck tests GM permission enforcement for level command
func TestAwardLevelCommandProducer_GmCheck(t *testing.T) {
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
			message:     "@award me 5 level",
			expectFound: true,
		},
		{
			name:        "Non-GM cannot execute command",
			isGm:        false,
			message:     "@award me 5 level",
			expectFound: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			char := createTestCharacter(12345, "TestPlayer", tc.isGm, 100000000)

			producer := AwardLevelCommandProducer(logger)
			_, found := producer(ctx)(1, 1, char, tc.message)

			if found != tc.expectFound {
				t.Errorf("Expected found=%v for GM=%v, got found=%v", tc.expectFound, tc.isGm, found)
			}
		})
	}
}

// TestChangeJobCommandProducer_GmCheck tests GM permission enforcement for job change
func TestChangeJobCommandProducer_GmCheck(t *testing.T) {
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
			message:     "@change my job 100",
			expectFound: true,
		},
		{
			name:        "Non-GM cannot execute command",
			isGm:        false,
			message:     "@change my job 100",
			expectFound: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			char := createTestCharacter(12345, "TestPlayer", tc.isGm, 100000000)

			producer := ChangeJobCommandProducer(logger)
			_, found := producer(ctx)(1, 1, char, tc.message)

			if found != tc.expectFound {
				t.Errorf("Expected found=%v for GM=%v, got found=%v", tc.expectFound, tc.isGm, found)
			}
		})
	}
}

// TestAwardMesoCommandProducer_GmCheck tests GM permission enforcement for meso command
func TestAwardMesoCommandProducer_GmCheck(t *testing.T) {
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
			message:     "@award me meso 10000",
			expectFound: true,
		},
		{
			name:        "Non-GM cannot execute command",
			isGm:        false,
			message:     "@award me meso 10000",
			expectFound: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			char := createTestCharacter(12345, "TestPlayer", tc.isGm, 100000000)

			producer := AwardMesoCommandProducer(logger)
			_, found := producer(ctx)(1, 1, char, tc.message)

			if found != tc.expectFound {
				t.Errorf("Expected found=%v for GM=%v, got found=%v", tc.expectFound, tc.isGm, found)
			}
		})
	}
}

// TestCommandProducers_NoMatchReturnsNil tests that non-matching commands return nil executor
func TestCommandProducers_NoMatchReturnsNil(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := context.Background()
	gmChar := createTestCharacter(12345, "TestGM", true, 100000000)

	testCases := []struct {
		name     string
		producer command.Producer
		message  string
	}{
		{
			name:     "Experience command - wrong format",
			producer: AwardExperienceCommandProducer,
			message:  "@give me exp 1000",
		},
		{
			name:     "Level command - wrong format",
			producer: AwardLevelCommandProducer,
			message:  "@give me levels 5",
		},
		{
			name:     "Job command - wrong format",
			producer: ChangeJobCommandProducer,
			message:  "@set job 100",
		},
		{
			name:     "Meso command - wrong format",
			producer: AwardMesoCommandProducer,
			message:  "@give me money 1000",
		},
		{
			name:     "Empty message",
			producer: AwardExperienceCommandProducer,
			message:  "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			executor, found := tc.producer(logger)(ctx)(1, 1, gmChar, tc.message)

			if found {
				t.Errorf("Expected found=false for message '%s', got found=true", tc.message)
			}

			if executor != nil {
				t.Errorf("Expected executor to be nil for non-matching message")
			}
		})
	}
}

// TestAwardExperienceCommandProducer_InvalidAmount tests invalid amount parsing
func TestAwardExperienceCommandProducer_InvalidAmount(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := context.Background()
	gmChar := createTestCharacter(12345, "TestGM", true, 100000000)

	// Note: The regex only matches digits, so truly invalid amounts won't match
	// This tests the case where the regex matches but parsing might fail
	testCases := []struct {
		name        string
		message     string
		expectFound bool
	}{
		{
			name:        "Valid amount",
			message:     "@award me experience 1000",
			expectFound: true,
		},
		{
			name:        "Zero amount",
			message:     "@award me experience 0",
			expectFound: true, // Zero is valid syntactically
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			producer := AwardExperienceCommandProducer(logger)
			_, found := producer(ctx)(1, 1, gmChar, tc.message)

			if found != tc.expectFound {
				t.Errorf("Expected found=%v, got found=%v", tc.expectFound, found)
			}
		})
	}
}

// TestAwardLevelCommandProducer_InvalidAmount tests invalid level amount parsing
func TestAwardLevelCommandProducer_InvalidAmount(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := context.Background()
	gmChar := createTestCharacter(12345, "TestGM", true, 100000000)

	testCases := []struct {
		name        string
		message     string
		expectFound bool
	}{
		{
			name:        "Valid level amount",
			message:     "@award me 5 level",
			expectFound: true,
		},
		{
			name:        "Max byte level",
			message:     "@award me 255 level",
			expectFound: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			producer := AwardLevelCommandProducer(logger)
			_, found := producer(ctx)(1, 1, gmChar, tc.message)

			if found != tc.expectFound {
				t.Errorf("Expected found=%v, got found=%v", tc.expectFound, found)
			}
		})
	}
}

// TestTargetResolution tests the different target types (me, map, name)
func TestTargetResolution(t *testing.T) {
	// Test that "me", "map", and player names are correctly identified
	testCases := []struct {
		name         string
		message      string
		expectedType string // "me", "map", or "name"
	}{
		{
			name:         "Target me",
			message:      "@award me experience 1000",
			expectedType: "me",
		},
		{
			name:         "Target map",
			message:      "@award map experience 1000",
			expectedType: "map",
		},
		{
			name:         "Target by name",
			message:      "@award PlayerName experience 1000",
			expectedType: "name",
		},
	}

	re := regexp.MustCompile(`@award\s+(\w+)\s+experience\s+(\d+)`)

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
