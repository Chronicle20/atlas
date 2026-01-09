package conversation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConditionModel_ReferenceIdRaw(t *testing.T) {
	tests := []struct {
		name        string
		referenceId string
		expected    string
	}{
		{
			name:        "static numeric value",
			referenceId: "4001261",
			expected:    "4001261",
		},
		{
			name:        "context reference curly brace format",
			referenceId: "{context.selectedItem}",
			expected:    "{context.selectedItem}",
		},
		{
			name:        "context reference legacy format",
			referenceId: "context.itemId",
			expected:    "context.itemId",
		},
		{
			name:        "empty string",
			referenceId: "",
			expected:    "",
		},
		{
			name:        "zero value",
			referenceId: "0",
			expected:    "0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			condition, err := NewConditionBuilder().
				SetType("item").
				SetOperator(">=").
				SetValue("1").
				SetReferenceId(tt.referenceId).
				Build()

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, condition.ReferenceIdRaw())
		})
	}
}

func TestConditionModel_ReferenceId_StaticValue(t *testing.T) {
	// Test that static numeric values still work with ReferenceId()
	condition, err := NewConditionBuilder().
		SetType("item").
		SetOperator(">=").
		SetValue("1").
		SetReferenceId("4001261").
		Build()

	assert.NoError(t, err)
	assert.Equal(t, uint32(4001261), condition.ReferenceId())
	assert.Equal(t, "4001261", condition.ReferenceIdRaw())
}

func TestConditionModel_ReferenceId_ContextReference_ReturnsZero(t *testing.T) {
	// Test that ReferenceId() returns 0 for context references (expected behavior)
	// The context resolution happens in the evaluator, not in the model
	condition, err := NewConditionBuilder().
		SetType("item").
		SetOperator(">=").
		SetValue("1").
		SetReferenceId("{context.selectedItem}").
		Build()

	assert.NoError(t, err)
	// ReferenceId() can't parse the context reference, so returns 0
	assert.Equal(t, uint32(0), condition.ReferenceId())
	// ReferenceIdRaw() returns the raw string
	assert.Equal(t, "{context.selectedItem}", condition.ReferenceIdRaw())
}

func TestConditionModel_ReferenceId_EmptyString(t *testing.T) {
	condition, err := NewConditionBuilder().
		SetType("item").
		SetOperator(">=").
		SetValue("1").
		SetReferenceId("").
		Build()

	assert.NoError(t, err)
	assert.Equal(t, uint32(0), condition.ReferenceId())
	assert.Equal(t, "", condition.ReferenceIdRaw())
}
