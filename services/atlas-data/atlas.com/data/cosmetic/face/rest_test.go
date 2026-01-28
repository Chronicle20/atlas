package face

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRestModel_GetName(t *testing.T) {
	m := RestModel{Id: 20000, Cash: false}
	assert.Equal(t, "faces", m.GetName())
}

func TestRestModel_GetID(t *testing.T) {
	tests := []struct {
		name     string
		id       uint32
		expected string
	}{
		{"standard face", 20000, "20000"},
		{"cash face", 20100, "20100"},
		{"high id", 29999, "29999"},
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
		{"valid face id", "20000", 20000, false},
		{"cash face id", "20100", 20100, false},
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
	cashFace := RestModel{Id: 20100, Cash: true}
	assert.True(t, cashFace.Cash)

	normalFace := RestModel{Id: 20000, Cash: false}
	assert.False(t, normalFace.Cash)
}

func TestRestModel_GetReferences(t *testing.T) {
	m := RestModel{Id: 20000}
	refs := m.GetReferences()
	assert.Empty(t, refs)
}

func TestRestModel_GetReferencedIDs(t *testing.T) {
	m := RestModel{Id: 20000}
	refIds := m.GetReferencedIDs()
	assert.Empty(t, refIds)
}

func TestRestModel_GetReferencedStructs(t *testing.T) {
	m := RestModel{Id: 20000}
	structs := m.GetReferencedStructs()
	assert.Empty(t, structs)
}
