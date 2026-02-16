package condition

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuilder_Valid(t *testing.T) {
	m, err := NewBuilder().
		SetType("item").
		SetOperator(">=").
		SetValue(10).
		SetReferenceId(1001).
		Build()

	require.NoError(t, err)
	assert.Equal(t, "item", m.Type())
	assert.Equal(t, ">=", m.Operator())
	assert.Equal(t, uint32(10), m.Value())
	assert.Equal(t, uint32(1001), m.ReferenceId())
}

func TestBuilder_TypeRequired(t *testing.T) {
	_, err := NewBuilder().
		SetOperator(">=").
		SetValue(10).
		Build()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "type is required")
}

func TestBuilder_OperatorRequired(t *testing.T) {
	_, err := NewBuilder().
		SetType("item").
		SetValue(10).
		Build()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "operator is required")
}

func TestBuilder_ZeroValueDefaults(t *testing.T) {
	m, err := NewBuilder().
		SetType("monster_kill").
		SetOperator("=").
		Build()

	require.NoError(t, err)
	assert.Equal(t, uint32(0), m.Value())
	assert.Equal(t, uint32(0), m.ReferenceId())
}
