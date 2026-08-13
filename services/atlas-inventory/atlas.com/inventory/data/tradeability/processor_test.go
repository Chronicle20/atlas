package tradeability

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
)

// TestExtractCarriesBothFields proves the shared Extract does not silently drop
// one of the two fields the karma gates need.
func TestExtractCarriesBothFields(t *testing.T) {
	m, err := extract(EquipmentRestModel{Id: 1002357, TradeBlock: true, TradeAvailable: 1})
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if !m.TradeBlock() {
		t.Fatal("TradeBlock() = false, want true")
	}
	if m.TradeAvailable() != 1 {
		t.Fatalf("TradeAvailable() = %d, want 1", m.TradeAvailable())
	}
}

// TestByIdProviderRejectsUnknownCompartment: an unknown compartment must be an
// error the caller refuses on, never a zero-valued permissive default.
func TestByIdProviderRejectsUnknownCompartment(t *testing.T) {
	p := &ProcessorImpl{}
	if _, err := p.ByIdProvider(inventory.Type(99), 1002357)(); err == nil {
		t.Fatal("ByIdProvider(99) returned no error; an unknown compartment must refuse")
	}
}
