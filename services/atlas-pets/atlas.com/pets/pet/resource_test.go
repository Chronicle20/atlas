package pet

import "testing"

func TestCreatePetName(t *testing.T) {
	// A provided name is preserved.
	if got := createPetName("Fluffy"); got != "Fluffy" {
		t.Fatalf("createPetName(\"Fluffy\") = %q, want %q", got, "Fluffy")
	}
	// An empty name (e.g. a pet granted via the inventory/award path, which
	// supplies no name) falls back to "Pet" so atlas-pets' "name is required"
	// check passes. A new pet's name would normally be the item's WZ name, but
	// atlas-data does not serve pet names today, so "Pet" is the effective
	// default (matching the cash-shop path).
	if got := createPetName(""); got != "Pet" {
		t.Fatalf("createPetName(\"\") = %q, want %q", got, "Pet")
	}
}
