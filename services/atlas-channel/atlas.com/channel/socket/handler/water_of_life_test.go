package handler

import (
	"atlas-channel/asset"
	"atlas-channel/pet"
	"testing"
	"time"

	"github.com/google/uuid"
)

func buildPet(t *testing.T, id uint32, expiration time.Time) pet.Model {
	t.Helper()
	m, err := pet.NewBuilder(id, 0, 5000000, "Pet").SetExpiration(expiration).Build()
	if err != nil {
		t.Fatalf("failed to build pet [%d]: %v", id, err)
	}
	return m
}

func TestSelectRevivableTargetPicksLatestPastExpiration(t *testing.T) {
	now := time.Now()
	// Two dolls; the one that dried up MOST RECENTLY wins (FR-4.2).
	older := buildPet(t, 1, now.Add(-72*time.Hour))
	newer := buildPet(t, 2, now.Add(-1*time.Hour))
	live := buildPet(t, 3, now.Add(24*time.Hour))

	got, ok := selectRevivableTarget([]pet.Model{older, live, newer}, now)
	if !ok {
		t.Fatal("expected a target")
	}
	if got.Id() != 2 {
		t.Fatalf("selected pet %d, want 2 (latest past expiration)", got.Id())
	}
}

func TestSelectRevivableTargetPicksLatestPastExpirationRegardlessOfOrder(t *testing.T) {
	now := time.Now()
	// Same trio as above, but the most-recently-expired pet is NOT last in the
	// input slice. A buggy implementation that just returned the last
	// surviving candidate (candidates[len(candidates)-1]) without comparing
	// expirations would pick "older" here and fail -- this pins the primary
	// sort key, not just the tiebreak.
	older := buildPet(t, 1, now.Add(-72*time.Hour))
	newer := buildPet(t, 2, now.Add(-1*time.Hour))
	live := buildPet(t, 3, now.Add(24*time.Hour))

	got, ok := selectRevivableTarget([]pet.Model{newer, older, live}, now)
	if !ok {
		t.Fatal("expected a target")
	}
	if got.Id() != 2 {
		t.Fatalf("selected pet %d, want 2 (latest past expiration)", got.Id())
	}
}

func TestSelectRevivableTargetTieBreaksOnLowestId(t *testing.T) {
	now := time.Now()
	exp := now.Add(-1 * time.Hour)
	// Two pets bought in one transaction share an expiration -- the norm, not
	// an edge case. The choice must be reproducible.
	got, ok := selectRevivableTarget([]pet.Model{buildPet(t, 9, exp), buildPet(t, 4, exp)}, now)
	if !ok {
		t.Fatal("expected a target")
	}
	if got.Id() != 4 {
		t.Fatalf("selected pet %d, want 4 (lowest id on a tie)", got.Id())
	}
}

func TestSelectRevivableTargetIgnoresLiveAndPermanentPets(t *testing.T) {
	now := time.Now()
	live := buildPet(t, 1, now.Add(24*time.Hour))
	permanent := buildPet(t, 2, time.Time{})

	if _, ok := selectRevivableTarget([]pet.Model{live, permanent}, now); ok {
		t.Fatal("expected no target when no pet has dried up")
	}
}

func TestSelectRevivableTargetEmpty(t *testing.T) {
	if _, ok := selectRevivableTarget(nil, time.Now()); ok {
		t.Fatal("expected no target for a character with no pets")
	}
}

func buildCashAsset(t *testing.T, id uint32, templateId uint32, slot int16) asset.Model {
	t.Helper()
	m, err := asset.NewBuilder(uuid.New(), templateId).SetId(id).SetSlot(slot).Build()
	if err != nil {
		t.Fatalf("failed to build asset [%d]: %v", id, err)
	}
	return m
}

func TestFindWaterOfLifePicksLowestSlot(t *testing.T) {
	// Two classification-518 assets: the lower slot wins, so the choice is
	// reproducible regardless of the backing slice's order.
	assets := []asset.Model{
		buildCashAsset(t, 2, 5180000, 9),
		buildCashAsset(t, 1, 5180000, 3),
	}
	got, ok := findWaterOfLife(assets)
	if !ok {
		t.Fatal("expected to find a Water of Life")
	}
	if got != 5180000 {
		t.Fatalf("templateId = %d, want 5180000", got)
	}
}

func TestFindWaterOfLifeIgnoresOtherCashClassifications(t *testing.T) {
	// 517 is a pet name tag and 519 a pet skill item; neither is a Water of Life.
	assets := []asset.Model{
		buildCashAsset(t, 1, 5170000, 1),
		buildCashAsset(t, 2, 5190000, 2),
		buildCashAsset(t, 3, 5000000, 3),
	}
	if _, ok := findWaterOfLife(assets); ok {
		t.Fatal("expected no Water of Life among neighbouring cash classifications")
	}
}

func TestFindWaterOfLifeNoneHeld(t *testing.T) {
	if _, ok := findWaterOfLife(nil); ok {
		t.Fatal("expected no Water of Life in an empty compartment")
	}
}
