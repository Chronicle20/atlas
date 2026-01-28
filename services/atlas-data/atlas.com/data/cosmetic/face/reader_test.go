package face

import (
	"atlas-data/xml"
	"testing"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
)

func TestParseFaceId_ValidFile(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		expected uint32
	}{
		{"standard face", "20000.img", 20000},
		{"cash face", "20100.img", 20100},
		{"female face", "21000.img", 21000},
		{"high id", "29999.img", 29999},
		{"with path", "/path/to/Face/20000.img", 20000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseFaceId(tt.filePath)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseFaceId_InvalidFile(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
	}{
		{"wrong extension", "20000.xml"},
		{"no extension", "20000"},
		{"invalid number", "abc.img"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseFaceId(tt.filePath)
			assert.Error(t, err)
		})
	}
}

func TestRead_BasicFace(t *testing.T) {
	l, _ := test.NewNullLogger()

	// Create a minimal XML node representing a face file
	node := xml.Node{
		Name:       "20000.img",
		ChildNodes: []xml.Node{},
	}

	provider := Read(l)(model.FixedProvider(node))
	result, err := provider()

	assert.NoError(t, err)
	assert.Equal(t, uint32(20000), result.Id)
	assert.False(t, result.Cash)
}

func TestRead_CashFace(t *testing.T) {
	l, _ := test.NewNullLogger()

	// Create XML node with cash info
	infoNode := xml.Node{
		Name: "info",
		IntegerNodes: []xml.IntegerNode{
			{Name: "cash", Value: "1"},
		},
	}
	node := xml.Node{
		Name:       "20100.img",
		ChildNodes: []xml.Node{infoNode},
	}

	provider := Read(l)(model.FixedProvider(node))
	result, err := provider()

	assert.NoError(t, err)
	assert.Equal(t, uint32(20100), result.Id)
	assert.True(t, result.Cash)
}

func TestRead_NonCashFace(t *testing.T) {
	l, _ := test.NewNullLogger()

	// Create XML node with cash=0 info
	infoNode := xml.Node{
		Name: "info",
		IntegerNodes: []xml.IntegerNode{
			{Name: "cash", Value: "0"},
		},
	}
	node := xml.Node{
		Name:       "20000.img",
		ChildNodes: []xml.Node{infoNode},
	}

	provider := Read(l)(model.FixedProvider(node))
	result, err := provider()

	assert.NoError(t, err)
	assert.Equal(t, uint32(20000), result.Id)
	assert.False(t, result.Cash)
}

func TestRead_InvalidFileName(t *testing.T) {
	l, _ := test.NewNullLogger()

	node := xml.Node{
		Name:       "invalid.xml",
		ChildNodes: []xml.Node{},
	}

	provider := Read(l)(model.FixedProvider(node))
	_, err := provider()

	assert.Error(t, err)
}
