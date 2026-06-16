package pet

import (
	"testing"
	"time"
)

func TestCreatePetExpiration(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	// A zero/epoch expiration (the bare inventory/award path) defaults to
	// now + the standard pet lifespan, so the pet is not created already-expired
	// ("dried up").
	got := createPetExpiration(time.Time{}, now)
	want := now.Add(petLifespan)
	if !got.Equal(want) {
		t.Fatalf("createPetExpiration(zero) = %v, want %v", got, want)
	}
	// A provided (non-zero) expiration is preserved.
	future := now.Add(24 * time.Hour)
	if got := createPetExpiration(future, now); !got.Equal(future) {
		t.Fatalf("createPetExpiration(future) = %v, want %v", got, future)
	}
}

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

func TestCreatePetLevel(t *testing.T) {
	// A valid level (1-30) is preserved.
	if got := createPetLevel(15); got != 15 {
		t.Fatalf("createPetLevel(15) = %d, want 15", got)
	}
	// A bare create (level 0, e.g. via the inventory/award path) defaults to 1 so
	// the model's "level must be between 1 and 30" check passes; the processor
	// then applies the rest of the new-pet defaults (closeness 0, full fullness).
	if got := createPetLevel(0); got != 1 {
		t.Fatalf("createPetLevel(0) = %d, want 1", got)
	}
	// Out-of-range high also normalizes to 1.
	if got := createPetLevel(99); got != 1 {
		t.Fatalf("createPetLevel(99) = %d, want 1", got)
	}
}
