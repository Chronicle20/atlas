package hair

import (
	"atlas-data/xml"
	"testing"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
)

func TestParseHairId_ValidFile(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		expected uint32
	}{
		{"male hair base", "30000.img", 30000},
		{"male hair with color", "30067.img", 30067},
		{"female hair base", "31000.img", 31000},
		{"female hair with color", "31157.img", 31157},
		{"high id", "49999.img", 49999},
		{"with path", "/path/to/Hair/30000.img", 30000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseHairId(tt.filePath)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseHairId_InvalidFile(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
	}{
		{"wrong extension", "30000.xml"},
		{"no extension", "30000"},
		{"invalid number", "abc.img"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseHairId(tt.filePath)
			assert.Error(t, err)
		})
	}
}

func TestRead_BasicHair(t *testing.T) {
	l, _ := test.NewNullLogger()

	// Create a minimal XML node representing a hair file
	node := xml.Node{
		Name:       "30000.img",
		ChildNodes: []xml.Node{},
	}

	provider := Read(l)(model.FixedProvider(node))
	result, err := provider()

	assert.NoError(t, err)
	assert.Equal(t, uint32(30000), result.Id)
	assert.False(t, result.Cash)
}

func TestRead_CashHair(t *testing.T) {
	l, _ := test.NewNullLogger()

	// Create XML node with cash info
	infoNode := xml.Node{
		Name: "info",
		IntegerNodes: []xml.IntegerNode{
			{Name: "cash", Value: "1"},
		},
	}
	node := xml.Node{
		Name:       "30100.img",
		ChildNodes: []xml.Node{infoNode},
	}

	provider := Read(l)(model.FixedProvider(node))
	result, err := provider()

	assert.NoError(t, err)
	assert.Equal(t, uint32(30100), result.Id)
	assert.True(t, result.Cash)
}

func TestRead_NonCashHair(t *testing.T) {
	l, _ := test.NewNullLogger()

	// Create XML node with cash=0 info
	infoNode := xml.Node{
		Name: "info",
		IntegerNodes: []xml.IntegerNode{
			{Name: "cash", Value: "0"},
		},
	}
	node := xml.Node{
		Name:       "30000.img",
		ChildNodes: []xml.Node{infoNode},
	}

	provider := Read(l)(model.FixedProvider(node))
	result, err := provider()

	assert.NoError(t, err)
	assert.Equal(t, uint32(30000), result.Id)
	assert.False(t, result.Cash)
}

func TestRead_HairWithColor(t *testing.T) {
	l, _ := test.NewNullLogger()

	// Hair 30067 = base 3006 with color 7
	node := xml.Node{
		Name:       "30067.img",
		ChildNodes: []xml.Node{},
	}

	provider := Read(l)(model.FixedProvider(node))
	result, err := provider()

	assert.NoError(t, err)
	assert.Equal(t, uint32(30067), result.Id)
}

func TestRead_FemaleHair(t *testing.T) {
	l, _ := test.NewNullLogger()

	node := xml.Node{
		Name:       "31150.img",
		ChildNodes: []xml.Node{},
	}

	provider := Read(l)(model.FixedProvider(node))
	result, err := provider()

	assert.NoError(t, err)
	assert.Equal(t, uint32(31150), result.Id)
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
