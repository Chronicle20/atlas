package conversation

import (
	"testing"

	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
)

func TestEvaluateContextValueAsInt_EmbeddedNegation(t *testing.T) {
	// This test covers the bug where "-{context.cost}" was passed to arithmetic
	// evaluation without resolving the {context.cost} placeholder first.
	l, _ := test.NewNullLogger()
	l.SetLevel(logrus.DebugLevel)

	var tm tenant.Model
	characterId := uint32(1)

	// Seed the registry with a conversation context containing the "cost" key
	ctx := NewConversationContextBuilder().
		SetCharacterId(characterId).
		AddContextValue("cost", "1000").
		Build()
	GetRegistry().SetContext(tm, characterId, ctx)
	defer GetRegistry().ClearContext(tm, characterId)

	executor := &OperationExecutorImpl{
		l: l,
		t: tm,
	}

	tests := []struct {
		name        string
		value       string
		expected    int
		expectError bool
	}{
		{
			name:     "direct context reference",
			value:    "{context.cost}",
			expected: 1000,
		},
		{
			name:     "negated context reference",
			value:    "-{context.cost}",
			expected: -1000,
		},
		{
			name:     "literal negative number",
			value:    "-500",
			expected: -500,
		},
		{
			name:     "literal positive number",
			value:    "200",
			expected: 200,
		},
		{
			name:        "missing context key",
			value:       "-{context.missing}",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := executor.evaluateContextValueAsInt(characterId, "amount", tt.value)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none, result: %d", result)
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if result != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, result)
			}
		})
	}
}
