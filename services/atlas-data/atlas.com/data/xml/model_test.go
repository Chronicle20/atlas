package xml

import (
	"testing"
)

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