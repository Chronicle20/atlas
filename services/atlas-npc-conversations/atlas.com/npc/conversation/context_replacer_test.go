package conversation

import (
	"testing"
)

func TestReplaceContextPlaceholders(t *testing.T) {
	tests := []struct {
		name        string
		text        string
		context     map[string]string
		expected    string
		expectError bool
	}{
		{
			name: "single placeholder",
			text: "You want #b#t{context.itemId}##k?",
			context: map[string]string{
				"itemId": "2000002",
			},
			expected:    "You want #b#t2000002##k?",
			expectError: false,
		},
		{
			name: "multiple placeholders",
			text: "Will you purchase #r{context.quantity}#k #b#t{context.itemId}##(s)#k? #t{context.itemId}# costs {context.price} mesos for one, so the total comes out to be #r{context.totalCost}#k mesos.",
			context: map[string]string{
				"itemId":    "2000002",
				"quantity":  "10",
				"price":     "310",
				"totalCost": "3100",
			},
			expected:    "Will you purchase #r10#k #b#t2000002##(s)#k? #t2000002# costs 310 mesos for one, so the total comes out to be #r3100#k mesos.",
			expectError: false,
		},
		{
			name: "no placeholders",
			text: "This is a normal text without any placeholders.",
			context: map[string]string{
				"itemId": "2000002",
			},
			expected:    "This is a normal text without any placeholders.",
			expectError: false,
		},
		{
			name: "missing context key",
			text: "You want {context.itemId}? It costs {context.price} mesos.",
			context: map[string]string{
				"itemId": "2000002",
			},
			expected:    "You want 2000002? It costs {context.price} mesos.",
			expectError: true,
		},
		{
			name:        "empty context",
			text:        "You want {context.itemId}?",
			context:     map[string]string{},
			expected:    "You want {context.itemId}?",
			expectError: true,
		},
		{
			name: "destination and cost placeholders",
			text: "Do you really want to move to #b#m{context.destination}##k? Well it'll cost you #b{context.cost} mesos#k.",
			context: map[string]string{
				"destination": "103000000",
				"cost":        "1000",
			},
			expected:    "Do you really want to move to #b#m103000000##k? Well it'll cost you #b1000 mesos#k.",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ReplaceContextPlaceholders(tt.text, tt.context)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if result != tt.expected {
				t.Errorf("Expected: %s\nGot: %s", tt.expected, result)
			}
		})
	}
}

func TestExtractContextValue(t *testing.T) {
	tests := []struct {
		name              string
		input             string
		context           map[string]string
		expectedValue     string
		expectedIsContext bool
		expectError       bool
	}{
		{
			name:  "curly brace format - found",
			input: "{context.itemId}",
			context: map[string]string{
				"itemId": "2000002",
			},
			expectedValue:     "2000002",
			expectedIsContext: true,
			expectError:       false,
		},
		{
			name:  "legacy format - found",
			input: "context.itemId",
			context: map[string]string{
				"itemId": "2000002",
			},
			expectedValue:     "2000002",
			expectedIsContext: true,
			expectError:       false,
		},
		{
			name:  "curly brace format - not found",
			input: "{context.itemId}",
			context: map[string]string{
				"price": "310",
			},
			expectedValue:     "",
			expectedIsContext: true,
			expectError:       true,
		},
		{
			name:  "legacy format - not found",
			input: "context.itemId",
			context: map[string]string{
				"price": "310",
			},
			expectedValue:     "",
			expectedIsContext: true,
			expectError:       true,
		},
		{
			name:              "literal value",
			input:             "2000002",
			context:           map[string]string{},
			expectedValue:     "2000002",
			expectedIsContext: false,
			expectError:       false,
		},
		{
			name:              "literal string",
			input:             "some literal text",
			context:           map[string]string{},
			expectedValue:     "some literal text",
			expectedIsContext: false,
			expectError:       false,
		},
		{
			name:  "numeric context value",
			input: "{context.totalCost}",
			context: map[string]string{
				"totalCost": "3100",
			},
			expectedValue:     "3100",
			expectedIsContext: true,
			expectError:       false,
		},
		{
			name:  "negative numeric context value",
			input: "{context.amount}",
			context: map[string]string{
				"amount": "-3100",
			},
			expectedValue:     "-3100",
			expectedIsContext: true,
			expectError:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, isContext, err := ExtractContextValue(tt.input, tt.context)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if value != tt.expectedValue {
				t.Errorf("Expected value: %s, got: %s", tt.expectedValue, value)
			}

			if isContext != tt.expectedIsContext {
				t.Errorf("Expected isContext: %v, got: %v", tt.expectedIsContext, isContext)
			}
		})
	}
}
