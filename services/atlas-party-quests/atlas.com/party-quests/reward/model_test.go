package reward

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuilder_Valid(t *testing.T) {
	m, err := NewBuilder().
		SetType("exp").
		SetAmount(5000).
		Build()

	require.NoError(t, err)
	assert.Equal(t, "exp", m.Type())
	assert.Equal(t, uint32(5000), m.Amount())
	assert.Empty(t, m.Items())
}

func TestBuilder_TypeRequired(t *testing.T) {
	_, err := NewBuilder().
		SetAmount(100).
		Build()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "type is required")
}

func TestBuilder_WithItems(t *testing.T) {
	wi, err := NewWeightedItemBuilder().
		SetTemplateId(2000001).
		SetWeight(50).
		SetQuantity(1).
		Build()
	require.NoError(t, err)

	m, err := NewBuilder().
		SetType("item").
		AddItem(wi).
		Build()

	require.NoError(t, err)
	assert.Len(t, m.Items(), 1)
	assert.Equal(t, uint32(2000001), m.Items()[0].TemplateId())
	assert.Equal(t, uint32(50), m.Items()[0].Weight())
	assert.Equal(t, uint32(1), m.Items()[0].Quantity())
}

func TestBuilder_SetItems(t *testing.T) {
	wi1, _ := NewWeightedItemBuilder().SetTemplateId(1001).SetWeight(30).SetQuantity(1).Build()
	wi2, _ := NewWeightedItemBuilder().SetTemplateId(1002).SetWeight(70).SetQuantity(2).Build()

	m, err := NewBuilder().
		SetType("item").
		SetItems([]WeightedItem{wi1, wi2}).
		Build()

	require.NoError(t, err)
	assert.Len(t, m.Items(), 2)
}

func TestWeightedItemBuilder_Valid(t *testing.T) {
	wi, err := NewWeightedItemBuilder().
		SetTemplateId(3000001).
		SetWeight(100).
		SetQuantity(5).
		Build()

	require.NoError(t, err)
	assert.Equal(t, uint32(3000001), wi.TemplateId())
	assert.Equal(t, uint32(100), wi.Weight())
	assert.Equal(t, uint32(5), wi.Quantity())
}

func TestWeightedItemBuilder_TemplateIdRequired(t *testing.T) {
	_, err := NewWeightedItemBuilder().
		SetWeight(50).
		SetQuantity(1).
		Build()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "templateId is required")
}
