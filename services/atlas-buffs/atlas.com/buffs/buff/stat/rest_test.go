package stat

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTransform(t *testing.T) {
	m := NewStat("STR", 10)

	rm, err := Transform(m)

	assert.NoError(t, err)
	assert.Equal(t, "STR", rm.Type)
	assert.Equal(t, int32(10), rm.Amount)
	assert.NotEmpty(t, rm.Id)
}

func TestTransform_DifferentStats(t *testing.T) {
	tests := []struct {
		name       string
		statType   string
		amount     int32
		expectType string
		expectAmt  int32
	}{
		{"STR stat", "STR", 10, "STR", 10},
		{"DEX stat", "DEX", 25, "DEX", 25},
		{"INT stat", "INT", 100, "INT", 100},
		{"LUK stat", "LUK", 50, "LUK", 50},
		{"Zero amount", "HP", 0, "HP", 0},
		{"Negative amount", "MP", -10, "MP", -10},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := NewStat(tc.statType, tc.amount)
			rm, err := Transform(m)

			assert.NoError(t, err)
			assert.Equal(t, tc.expectType, rm.Type)
			assert.Equal(t, tc.expectAmt, rm.Amount)
		})
	}
}

func TestRestModel_Fields(t *testing.T) {
	rm := RestModel{
		Type:   "STR",
		Amount: 10,
	}

	assert.Equal(t, "STR", rm.Type)
	assert.Equal(t, int32(10), rm.Amount)
}

func TestRestModel_GetName(t *testing.T) {
	rm := RestModel{}
	assert.Equal(t, "stats", rm.GetName())
}

func TestRestModel_GetID(t *testing.T) {
	rm := RestModel{Id: "test-id"}
	assert.Equal(t, "test-id", rm.GetID())
}

func TestRestModel_SetID(t *testing.T) {
	rm := &RestModel{}
	err := rm.SetID("new-id")

	assert.NoError(t, err)
	assert.Equal(t, "new-id", rm.Id)
}
