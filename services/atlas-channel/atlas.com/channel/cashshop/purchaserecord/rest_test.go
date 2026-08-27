package purchaserecord

import "testing"

func TestExtract(t *testing.T) {
	rm := RestModel{SerialNumber: 5555, Purchased: true, Count: 2}

	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if m.SerialNumber() != 5555 {
		t.Errorf("SerialNumber() = %d, want 5555", m.SerialNumber())
	}
	if !m.Purchased() {
		t.Error("Purchased() = false, want true")
	}
	if m.Count() != 2 {
		t.Errorf("Count() = %d, want 2", m.Count())
	}
}

func TestExtractInvalidSerialNumber(t *testing.T) {
	rm := RestModel{SerialNumber: 0, Purchased: true, Count: 2}

	_, err := Extract(rm)
	if err == nil {
		t.Fatal("Extract: expected error for zero serial number, got nil")
	}
}

func TestTransform(t *testing.T) {
	m := NewModelBuilder(5555).SetPurchased(true).SetCount(2).MustBuild()

	rm, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	if rm.SerialNumber != 5555 {
		t.Errorf("SerialNumber = %d, want 5555", rm.SerialNumber)
	}
	if !rm.Purchased {
		t.Error("Purchased = false, want true")
	}
	if rm.Count != 2 {
		t.Errorf("Count = %d, want 2", rm.Count)
	}
}
