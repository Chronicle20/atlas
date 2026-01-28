package hair

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRestModel_GetName(t *testing.T) {
	m := RestModel{Id: 30000, Cash: false}
	assert.Equal(t, "hairs", m.GetName())
}

func TestRestModel_GetID(t *testing.T) {
	tests := []struct {
		name     string
		id       uint32
		expected string
	}{
		{"male hair", 30000, "30000"},
		{"female hair", 31000, "31000"},
		{"cash hair", 30100, "30100"},
		{"hair with color", 30067, "30067"},
		{"high id", 49999, "49999"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := RestModel{Id: tt.id}
			assert.Equal(t, tt.expected, m.GetID())
		})
	}
}

func TestRestModel_SetID(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectedId  uint32
		expectError bool
	}{
		{"valid male hair id", "30000", 30000, false},
		{"valid female hair id", "31000", 31000, false},
		{"hair with color", "30067", 30067, false},
		{"invalid string", "abc", 0, true},
		{"empty string", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &RestModel{}
			err := m.SetID(tt.input)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedId, m.Id)
			}
		})
	}
}

func TestRestModel_CashField(t *testing.T) {
	cashHair := RestModel{Id: 30100, Cash: true}
	assert.True(t, cashHair.Cash)

	normalHair := RestModel{Id: 30000, Cash: false}
	assert.False(t, normalHair.Cash)
}

func TestRestModel_GetReferences(t *testing.T) {
	m := RestModel{Id: 30000}
	refs := m.GetReferences()
	assert.Empty(t, refs)
}

func TestRestModel_GetReferencedIDs(t *testing.T) {
	m := RestModel{Id: 30000}
	refIds := m.GetReferencedIDs()
	assert.Empty(t, refIds)
}

func TestRestModel_GetReferencedStructs(t *testing.T) {
	m := RestModel{Id: 30000}
	structs := m.GetReferencedStructs()
	assert.Empty(t, structs)
}
