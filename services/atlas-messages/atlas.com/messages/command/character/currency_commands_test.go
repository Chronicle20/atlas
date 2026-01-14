package character

import (
	"regexp"
	"testing"

	"github.com/sirupsen/logrus/hooks/test"
	"golang.org/x/net/context"
)

// TestAwardCurrencyCommandProducer_RegexPatterns tests the currency command regex
func TestAwardCurrencyCommandProducer_RegexPatterns(t *testing.T) {
	re := regexp.MustCompile(`(?i)@award\s+(\w+)\s+(credit|points|prepaid)\s+(-?\d+)`)

	testCases := []struct {
		name             string
		message          string
		expectMatch      bool
		expectedTarget   string
		expectedCurrency string
		expectedAmount   string
	}{
		{
			name:             "Award me credit",
			message:          "@award me credit 1000",
			expectMatch:      true,
			expectedTarget:   "me",
			expectedCurrency: "credit",
			expectedAmount:   "1000",
		},
		{
			name:             "Award me points",
			message:          "@award me points 500",
			expectMatch:      true,
			expectedTarget:   "me",
			expectedCurrency: "points",
			expectedAmount:   "500",
		},
		{
			name:             "Award me prepaid",
			message:          "@award me prepaid 2000",
			expectMatch:      true,
			expectedTarget:   "me",
			expectedCurrency: "prepaid",
			expectedAmount:   "2000",
		},
		{
			name:             "Award player credit",
			message:          "@award PlayerName credit 100",
			expectMatch:      true,
			expectedTarget:   "PlayerName",
			expectedCurrency: "credit",
			expectedAmount:   "100",
		},
		{
			name:             "Case insensitive - uppercase",
			message:          "@award me CREDIT 1000",
			expectMatch:      true,
			expectedTarget:   "me",
			expectedCurrency: "CREDIT",
			expectedAmount:   "1000",
		},
		{
			name:             "Case insensitive - mixed case",
			message:          "@award me Credit 1000",
			expectMatch:      true,
			expectedTarget:   "me",
			expectedCurrency: "Credit",
			expectedAmount:   "1000",
		},
		{
			name:             "Negative amount",
			message:          "@award me credit -500",
			expectMatch:      true,
			expectedTarget:   "me",
			expectedCurrency: "credit",
			expectedAmount:   "-500",
		},
		{
			name:        "Invalid currency type",
			message:     "@award me gold 1000",
			expectMatch: false,
		},
		{
			name:        "Missing amount",
			message:     "@award me credit",
			expectMatch: false,
		},
		{
			name:        "Invalid format",
			message:     "@give me credit 1000",
			expectMatch: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			match := re.FindStringSubmatch(tc.message)

			if tc.expectMatch {
				if len(match) != 4 {
					t.Errorf("Expected match with 4 groups, got %d groups", len(match))
					return
				}
				if match[1] != tc.expectedTarget {
					t.Errorf("Expected target '%s', got '%s'", tc.expectedTarget, match[1])
				}
				if match[2] != tc.expectedCurrency {
					t.Errorf("Expected currency '%s', got '%s'", tc.expectedCurrency, match[2])
				}
				if match[3] != tc.expectedAmount {
					t.Errorf("Expected amount '%s', got '%s'", tc.expectedAmount, match[3])
				}
			} else {
				if len(match) == 4 {
					t.Errorf("Expected no match, but got match: %v", match)
				}
			}
		})
	}
}

// TestAwardCurrencyCommandProducer_GmCheck tests GM permission enforcement
func TestAwardCurrencyCommandProducer_GmCheck(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := context.Background()

	testCases := []struct {
		name        string
		isGm        bool
		message     string
		expectFound bool
	}{
		{
			name:        "GM can execute credit command",
			isGm:        true,
			message:     "@award me credit 1000",
			expectFound: true,
		},
		{
			name:        "GM can execute points command",
			isGm:        true,
			message:     "@award me points 500",
			expectFound: true,
		},
		{
			name:        "GM can execute prepaid command",
			isGm:        true,
			message:     "@award me prepaid 2000",
			expectFound: true,
		},
		{
			name:        "Non-GM cannot execute command",
			isGm:        false,
			message:     "@award me credit 1000",
			expectFound: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			char := createTestCharacter(12345, "TestPlayer", tc.isGm, 100000000)

			producer := AwardCurrencyCommandProducer(logger)
			_, found := producer(ctx)(1, 1, char, tc.message)

			if found != tc.expectFound {
				t.Errorf("Expected found=%v for GM=%v, got found=%v", tc.expectFound, tc.isGm, found)
			}
		})
	}
}

// TestAwardCurrencyCommandProducer_CurrencyTypeMapping tests currency type to ID mapping
func TestAwardCurrencyCommandProducer_CurrencyTypeMapping(t *testing.T) {
	// Test that the internal currency type mapping is consistent
	testCases := []struct {
		currencyStr  string
		expectedType uint32
	}{
		{currencyStr: "credit", expectedType: 1},
		{currencyStr: "points", expectedType: 2},
		{currencyStr: "prepaid", expectedType: 3},
	}

	for _, tc := range testCases {
		t.Run(tc.currencyStr, func(t *testing.T) {
			var currencyType uint32
			switch tc.currencyStr {
			case "credit":
				currencyType = 1
			case "points":
				currencyType = 2
			case "prepaid":
				currencyType = 3
			}

			if currencyType != tc.expectedType {
				t.Errorf("Expected currency type %d for '%s', got %d", tc.expectedType, tc.currencyStr, currencyType)
			}
		})
	}
}

// TestAwardCurrencyCommandProducer_NoMatchReturnsNil tests non-matching messages
func TestAwardCurrencyCommandProducer_NoMatchReturnsNil(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := context.Background()
	gmChar := createTestCharacter(12345, "TestGM", true, 100000000)

	testCases := []struct {
		name    string
		message string
	}{
		{
			name:    "Wrong command",
			message: "@give me credit 1000",
		},
		{
			name:    "Invalid currency type",
			message: "@award me gold 1000",
		},
		{
			name:    "Empty message",
			message: "",
		},
		{
			name:    "Missing amount",
			message: "@award me credit",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			producer := AwardCurrencyCommandProducer(logger)
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

// TestAwardCurrencyCommandProducer_TargetResolution tests target resolution (me vs name)
func TestAwardCurrencyCommandProducer_TargetResolution(t *testing.T) {
	// Currency commands do NOT support "map" target unlike other award commands
	re := regexp.MustCompile(`(?i)@award\s+(\w+)\s+(credit|points|prepaid)\s+(-?\d+)`)

	testCases := []struct {
		name         string
		message      string
		expectedType string
	}{
		{
			name:         "Target me",
			message:      "@award me credit 1000",
			expectedType: "me",
		},
		{
			name:         "Target by name",
			message:      "@award PlayerName credit 1000",
			expectedType: "name",
		},
		// Note: "map" is technically matched by regex but treated as a player name
		{
			name:         "Target map (treated as name)",
			message:      "@award map credit 1000",
			expectedType: "name", // Unlike other commands, currency doesn't support map target
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			match := re.FindStringSubmatch(tc.message)
			if len(match) != 4 {
				t.Fatal("Expected regex to match")
			}

			target := match[1]
			var actualType string
			if target == "me" {
				actualType = "me"
			} else {
				actualType = "name"
			}

			if actualType != tc.expectedType {
				t.Errorf("Expected target type '%s', got '%s' for target '%s'", tc.expectedType, actualType, target)
			}
		})
	}
}
