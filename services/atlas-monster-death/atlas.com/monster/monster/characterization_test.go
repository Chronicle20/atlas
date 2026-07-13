package monster

import (
	"math"
	"testing"

	"atlas-monster-death/monster/drop"
)

func TestCalculateExperienceStandardDeviationThreshold_Uniform(t *testing.T) {
	// identical ratios → variance 0 → threshold == mean
	got := calculateExperienceStandardDeviationThreshold([]float64{0.25, 0.25, 0.25, 0.25}, 4)
	if math.Abs(got-0.25) > 1e-9 {
		t.Fatalf("expected 0.25, got %f", got)
	}
}

func TestCalculateExperienceStandardDeviationThreshold_Skewed(t *testing.T) {
	// ratios {0.7,0.1,0.1,0.1}, totalEntries 4: mean=0.25,
	// var=((0.45)^2+3*(0.15)^2)/4=0.0675, threshold=0.25+sqrt(0.0675)
	got := calculateExperienceStandardDeviationThreshold([]float64{0.7, 0.1, 0.1, 0.1}, 4)
	want := 0.25 + math.Sqrt(0.0675)
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("expected %f, got %f", want, got)
	}
}

func TestIsWhiteExperienceGain(t *testing.T) {
	ratios := map[uint32]float64{1: 0.6, 2: 0.2}
	if !isWhiteExperienceGain(1, ratios, 0.5) {
		t.Fatal("ratio above threshold should be white gain")
	}
	if isWhiteExperienceGain(2, ratios, 0.5) {
		t.Fatal("ratio below threshold should not be white gain")
	}
	if isWhiteExperienceGain(99, ratios, 0.5) {
		t.Fatal("absent character should not be white gain")
	}
}

func TestGetSuccessfulDrops_DeterministicEdges(t *testing.T) {
	// evaluateSuccess: rand.Int31n(999999) < min(chance*rate, MaxInt32)
	certain, _ := drop.NewBuilder().SetItemId(1000).SetChance(999999).SetMinimumQuantity(1).SetMaximumQuantity(1).Build()
	never, _ := drop.NewBuilder().SetItemId(2000).SetChance(0).SetMinimumQuantity(1).SetMaximumQuantity(1).Build()
	for i := 0; i < 100; i++ {
		got := getSuccessfulDrops([]drop.Model{certain, never}, 1.0)
		if len(got) != 1 || got[0].ItemId() != 1000 {
			t.Fatalf("iteration %d: expected exactly the certain drop, got %d entries", i, len(got))
		}
	}
}
