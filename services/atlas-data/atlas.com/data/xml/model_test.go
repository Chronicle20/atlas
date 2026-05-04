package xml

import (
	"testing"
)

func TestGetShort_StringFallback(t *testing.T) {
	tests := []struct {
		name        string
		integerNodes []IntegerNode
		stringNodes  []StringNode
		field       string
		def         uint16
		want        uint16
	}{
		{
			name:        "string-only matches and parses",
			stringNodes: []StringNode{{Name: "reqLUK", Value: "120"}},
			field:       "reqLUK",
			def:         0,
			want:        120,
		},
		{
			name:         "int wins over string",
			integerNodes: []IntegerNode{{Name: "reqLUK", Value: "100"}},
			stringNodes:  []StringNode{{Name: "reqLUK", Value: "120"}},
			field:        "reqLUK",
			def:          0,
			want:         100,
		},
		{
			name:        "unparseable string returns default",
			stringNodes: []StringNode{{Name: "reqLUK", Value: "abc"}},
			field:       "reqLUK",
			def:         42,
			want:        42,
		},
		{
			name:  "no match returns default",
			field: "reqLUK",
			def:   7,
			want:  7,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			n := &Node{IntegerNodes: tc.integerNodes, StringNodes: tc.stringNodes}
			if got := n.GetShort(tc.field, tc.def); got != tc.want {
				t.Errorf("GetShort(%q) = %d, want %d", tc.field, got, tc.want)
			}
		})
	}
}

func TestGetBool_StringFallback(t *testing.T) {
	tests := []struct {
		name         string
		integerNodes []IntegerNode
		stringNodes  []StringNode
		field        string
		def          bool
		want         bool
	}{
		{
			name:        "string-only true",
			stringNodes: []StringNode{{Name: "cash", Value: "1"}},
			field:       "cash",
			def:         false,
			want:        true,
		},
		{
			name:        "string-only false",
			stringNodes: []StringNode{{Name: "cash", Value: "0"}},
			field:       "cash",
			def:         true,
			want:        false,
		},
		{
			name:         "int wins over string",
			integerNodes: []IntegerNode{{Name: "cash", Value: "0"}},
			stringNodes:  []StringNode{{Name: "cash", Value: "1"}},
			field:        "cash",
			def:          true,
			want:         false,
		},
		{
			name:        "unparseable string returns default",
			stringNodes: []StringNode{{Name: "cash", Value: "yes"}},
			field:       "cash",
			def:         true,
			want:        true,
		},
		{
			name:  "no match returns default",
			field: "cash",
			def:   true,
			want:  true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			n := &Node{IntegerNodes: tc.integerNodes, StringNodes: tc.stringNodes}
			if got := n.GetBool(tc.field, tc.def); got != tc.want {
				t.Errorf("GetBool(%q) = %v, want %v", tc.field, got, tc.want)
			}
		})
	}
}

func TestGetIntegerWithDefault_StringFallback(t *testing.T) {
	tests := []struct {
		name         string
		integerNodes []IntegerNode
		stringNodes  []StringNode
		field        string
		def          int32
		want         int32
	}{
		{
			name:        "string-only matches and parses",
			stringNodes: []StringNode{{Name: "reqLevel", Value: "35"}},
			field:       "reqLevel",
			def:         0,
			want:        35,
		},
		{
			name:         "int wins over string",
			integerNodes: []IntegerNode{{Name: "reqLevel", Value: "118"}},
			stringNodes:  []StringNode{{Name: "reqLevel", Value: "35"}},
			field:        "reqLevel",
			def:          0,
			want:         118,
		},
		{
			name:        "unparseable string returns default",
			stringNodes: []StringNode{{Name: "reqLevel", Value: "abc"}},
			field:       "reqLevel",
			def:         99,
			want:        99,
		},
		{
			name:  "no match returns default",
			field: "reqLevel",
			def:   13,
			want:  13,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			n := &Node{IntegerNodes: tc.integerNodes, StringNodes: tc.stringNodes}
			if got := n.GetIntegerWithDefault(tc.field, tc.def); got != tc.want {
				t.Errorf("GetIntegerWithDefault(%q) = %d, want %d", tc.field, got, tc.want)
			}
		})
	}
}

func TestGetFloatWithDefault_StringFallback(t *testing.T) {
	tests := []struct {
		name         string
		integerNodes []IntegerNode
		stringNodes  []StringNode
		field        string
		def          float64
		want         float64
	}{
		{
			name:        "string-only matches and parses",
			stringNodes: []StringNode{{Name: "rate", Value: "1.5"}},
			field:       "rate",
			def:         0,
			want:        1.5,
		},
		{
			name:         "int wins over string",
			integerNodes: []IntegerNode{{Name: "rate", Value: "2"}},
			stringNodes:  []StringNode{{Name: "rate", Value: "1.5"}},
			field:        "rate",
			def:          0,
			want:         2,
		},
		{
			name:        "unparseable string returns default",
			stringNodes: []StringNode{{Name: "rate", Value: "abc"}},
			field:       "rate",
			def:         9.99,
			want:        9.99,
		},
		{
			name:  "no match returns default",
			field: "rate",
			def:   3.14,
			want:  3.14,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			n := &Node{IntegerNodes: tc.integerNodes, StringNodes: tc.stringNodes}
			if got := n.GetFloatWithDefault(tc.field, tc.def); got != tc.want {
				t.Errorf("GetFloatWithDefault(%q) = %f, want %f", tc.field, got, tc.want)
			}
		})
	}
}

func TestGetDouble(t *testing.T) {
	// Create a test Node with various DoubleNodes
	node := &Node{
		DoubleNodes: []DoubleNode{
			{Name: "periodDecimal", Value: "10.5"},
			{Name: "commaDecimal", Value: "20,75"},
			{Name: "invalidValue", Value: "not-a-number"},
		},
	}

	// Test case 1: Normal case with period decimal separator
	result1 := node.GetDouble("periodDecimal", 0.0)
	if result1 != 10.5 {
		t.Errorf("Expected 10.5 for periodDecimal, got %f", result1)
	}

	// Test case 2: Case with comma decimal separator
	result2 := node.GetDouble("commaDecimal", 0.0)
	if result2 != 20.75 {
		t.Errorf("Expected 20.75 for commaDecimal, got %f", result2)
	}

	// Test case 3: Case with invalid value (should return default)
	result3 := node.GetDouble("invalidValue", 99.9)
	if result3 != 99.9 {
		t.Errorf("Expected default value 99.9 for invalidValue, got %f", result3)
	}

	// Test case 4: Case where the node doesn't exist (should return default)
	result4 := node.GetDouble("nonExistentNode", 42.42)
	if result4 != 42.42 {
		t.Errorf("Expected default value 42.42 for nonExistentNode, got %f", result4)
	}
}

func TestGetDoubleFromXML(t *testing.T) {
	// Test with actual XML parsing
	xmlData := []byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="test">
  <double name="unitPrice" value="0.5"/>
  <double name="taxRate" value="7,5"/>
  <double name="invalidDouble" value="invalid"/>
</imgdir>`)

	provider := FromByteArrayProvider(xmlData)
	parsedNode, err := provider()
	
	if err != nil {
		t.Fatalf("Failed to parse XML: %v", err)
	}

	// Test parsed values
	unitPrice := parsedNode.GetDouble("unitPrice", 0.0)
	if unitPrice != 0.5 {
		t.Errorf("Expected unitPrice 0.5, got %f", unitPrice)
	}

	taxRate := parsedNode.GetDouble("taxRate", 0.0)
	if taxRate != 7.5 {
		t.Errorf("Expected taxRate 7.5, got %f", taxRate)
	}

	invalidDouble := parsedNode.GetDouble("invalidDouble", 123.45)
	if invalidDouble != 123.45 {
		t.Errorf("Expected default value 123.45 for invalidDouble, got %f", invalidDouble)
	}

	nonExistent := parsedNode.GetDouble("nonExistent", 99.99)
	if nonExistent != 99.99 {
		t.Errorf("Expected default value 99.99 for nonExistent, got %f", nonExistent)
	}
}