package item

import "testing"

// TestExtractParsesTemplateId asserts the JSON:API id string (the item template
// id) is parsed into the model's uint32 itemId and the name is carried through.
func TestExtractParsesTemplateId(t *testing.T) {
	m, err := Extract(RestModel{Id: "1302000", Name: "Sword"})
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}
	if m.ItemId() != 1302000 {
		t.Errorf("itemId: want 1302000, got %d", m.ItemId())
	}
	if m.Name() != "Sword" {
		t.Errorf("name: want %q, got %q", "Sword", m.Name())
	}
}

// TestExtractRejectsNonNumericId asserts a non-numeric id surfaces as an error
// rather than a silent zero id.
func TestExtractRejectsNonNumericId(t *testing.T) {
	if _, err := Extract(RestModel{Id: "not-an-id", Name: "x"}); err == nil {
		t.Errorf("Extract should error on a non-numeric id")
	}
}
