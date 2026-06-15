package pet

import "testing"

func TestCreatePetName(t *testing.T) {
	// A provided name is preserved.
	if got := createPetName("Fluffy"); got != "Fluffy" {
		t.Fatalf("createPetName(\"Fluffy\") = %q, want %q", got, "Fluffy")
	}
	// An empty name (e.g. a pet granted via the generic inventory/award path,
	// which supplies no name) falls back to "Pet" so the model's "name is
	// required" check passes. The player-facing cash-shop path resolves the WZ
	// name from atlas-data explicitly; the generic award path does not.
	if got := createPetName(""); got != "Pet" {
		t.Fatalf("createPetName(\"\") = %q, want %q", got, "Pet")
	}
}
