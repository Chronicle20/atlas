package monster

import (
	"atlas-monster-death/monster/drop"
	"math"
	"testing"
)

func TestEvaluateSuccess_ZeroChance(t *testing.T) {
	d, _ := drop.NewBuilder().SetChance(0).Build()

	// With 0 chance, should always fail
	for i := 0; i < 100; i++ {
		if evaluateSuccess(d, 1) {
			t.Errorf("evaluateSuccess should return false for 0%% chance")
		}
	}
}

func TestEvaluateSuccess_MaxChance(t *testing.T) {
	// Chance of 999999 means 100% success (rand.Int31n(999999) always < 999999)
	d, _ := drop.NewBuilder().SetChance(999999).Build()

	for i := 0; i < 100; i++ {
		if !evaluateSuccess(d, 1) {
			t.Errorf("evaluateSuccess should return true for 999999 chance (100%%)")
		}
	}
}

func TestEvaluateSuccess_FiftyPercentChance(t *testing.T) {
	// 50% chance = 499999
	d, _ := drop.NewBuilder().SetChance(499999).Build()

	successes := 0
	iterations := 10000

	for i := 0; i < iterations; i++ {
		if evaluateSuccess(d, 1) {
			successes++
		}
	}

	// With 50% chance, we expect roughly 50% success rate
	// Allow 5% tolerance
	successRate := float64(successes) / float64(iterations)
	if successRate < 0.45 || successRate > 0.55 {
		t.Errorf("Expected ~50%% success rate, got %.2f%%", successRate*100)
	}
}

func TestGetSuccessfulDrops_EmptyInput(t *testing.T) {
	result := getSuccessfulDrops([]drop.Model{}, 1)
	if len(result) != 0 {
		t.Errorf("Expected empty result for empty input, got %d items", len(result))
	}
}

func TestGetSuccessfulDrops_AllFail(t *testing.T) {
	d1, _ := drop.NewBuilder().SetItemId(1).SetChance(0).Build()
	d2, _ := drop.NewBuilder().SetItemId(2).SetChance(0).Build()
	d3, _ := drop.NewBuilder().SetItemId(3).SetChance(0).Build()
	drops := []drop.Model{d1, d2, d3}

	result := getSuccessfulDrops(drops, 1)
	if len(result) != 0 {
		t.Errorf("Expected empty result when all drops fail, got %d items", len(result))
	}
}

func TestGetSuccessfulDrops_AllSucceed(t *testing.T) {
	d1, _ := drop.NewBuilder().SetItemId(1).SetChance(999999).Build()
	d2, _ := drop.NewBuilder().SetItemId(2).SetChance(999999).Build()
	d3, _ := drop.NewBuilder().SetItemId(3).SetChance(999999).Build()
	drops := []drop.Model{d1, d2, d3}

	result := getSuccessfulDrops(drops, 1)
	if len(result) != 3 {
		t.Errorf("Expected 3 results when all drops succeed, got %d", len(result))
	}
}

func TestGetSuccessfulDrops_PreservesOrder(t *testing.T) {
	d1, _ := drop.NewBuilder().SetItemId(100).SetChance(999999).Build()
	d2, _ := drop.NewBuilder().SetItemId(200).SetChance(999999).Build()
	d3, _ := drop.NewBuilder().SetItemId(300).SetChance(999999).Build()
	drops := []drop.Model{d1, d2, d3}

	result := getSuccessfulDrops(drops, 1)
	if result[0].ItemId() != 100 || result[1].ItemId() != 200 || result[2].ItemId() != 300 {
		t.Errorf("Expected order preservation, got items %d, %d, %d",
			result[0].ItemId(), result[1].ItemId(), result[2].ItemId())
	}
}

func TestCalculateExperienceStandardDeviationThreshold_SingleEntry(t *testing.T) {
	ratios := []float64{0.5}
	result := calculateExperienceStandardDeviationThreshold(ratios, 1)

	// With single entry: average = 0.5, variance = 0, stddev = 0
	// Result = 0.5 + 0 = 0.5
	if math.Abs(result-0.5) > 0.0001 {
		t.Errorf("Expected 0.5, got %f", result)
	}
}

func TestCalculateExperienceStandardDeviationThreshold_TwoEqualEntries(t *testing.T) {
	ratios := []float64{0.5, 0.5}
	result := calculateExperienceStandardDeviationThreshold(ratios, 2)

	// With two equal entries: average = 0.5, variance = 0, stddev = 0
	// Result = 0.5 + 0 = 0.5
	if math.Abs(result-0.5) > 0.0001 {
		t.Errorf("Expected 0.5, got %f", result)
	}
}

func TestCalculateExperienceStandardDeviationThreshold_KnownDistribution(t *testing.T) {
	// Test with values 0.2, 0.3, 0.5 (total entries = 3)
	ratios := []float64{0.2, 0.3, 0.5}
	result := calculateExperienceStandardDeviationThreshold(ratios, 3)

	// Average = (0.2 + 0.3 + 0.5) / 3 = 1.0 / 3 = 0.333...
	// Variance = ((0.2-0.333)^2 + (0.3-0.333)^2 + (0.5-0.333)^2) / 3
	//          = (0.0178 + 0.0011 + 0.0278) / 3
	//          = 0.0467 / 3 = 0.01556
	// StdDev = sqrt(0.01556) = 0.1247
	// Result = 0.333 + 0.1247 = 0.458

	expected := 1.0/3.0 + math.Sqrt(((0.2-1.0/3.0)*(0.2-1.0/3.0)+(0.3-1.0/3.0)*(0.3-1.0/3.0)+(0.5-1.0/3.0)*(0.5-1.0/3.0))/3.0)

	if math.Abs(result-expected) > 0.0001 {
		t.Errorf("Expected %f, got %f", expected, result)
	}
}

func TestCalculateExperienceStandardDeviationThreshold_HighVariance(t *testing.T) {
	// Test with extreme values
	ratios := []float64{0.1, 0.9}
	result := calculateExperienceStandardDeviationThreshold(ratios, 2)

	// Average = 0.5
	// Variance = ((0.1-0.5)^2 + (0.9-0.5)^2) / 2 = (0.16 + 0.16) / 2 = 0.16
	// StdDev = sqrt(0.16) = 0.4
	// Result = 0.5 + 0.4 = 0.9
	expected := 0.9

	if math.Abs(result-expected) > 0.0001 {
		t.Errorf("Expected %f, got %f", expected, result)
	}
}

func TestIsWhiteExperienceGain_AboveThreshold(t *testing.T) {
	personalRatio := map[uint32]float64{
		1: 0.8,
		2: 0.2,
	}

	// Character 1 has ratio 0.8, threshold is 0.5 -> should be true
	if !isWhiteExperienceGain(1, personalRatio, 0.5) {
		t.Errorf("Expected true when ratio (0.8) >= threshold (0.5)")
	}
}

func TestIsWhiteExperienceGain_BelowThreshold(t *testing.T) {
	personalRatio := map[uint32]float64{
		1: 0.3,
		2: 0.7,
	}

	// Character 1 has ratio 0.3, threshold is 0.5 -> should be false
	if isWhiteExperienceGain(1, personalRatio, 0.5) {
		t.Errorf("Expected false when ratio (0.3) < threshold (0.5)")
	}
}

func TestIsWhiteExperienceGain_ExactlyAtThreshold(t *testing.T) {
	personalRatio := map[uint32]float64{
		1: 0.5,
	}

	// Character 1 has ratio 0.5, threshold is 0.5 -> should be true (>=)
	if !isWhiteExperienceGain(1, personalRatio, 0.5) {
		t.Errorf("Expected true when ratio equals threshold")
	}
}

func TestIsWhiteExperienceGain_CharacterNotInMap(t *testing.T) {
	personalRatio := map[uint32]float64{
		1: 0.8,
	}

	// Character 99 is not in map -> should be false
	if isWhiteExperienceGain(99, personalRatio, 0.5) {
		t.Errorf("Expected false when character not in personalRatio map")
	}
}

func TestIsWhiteExperienceGain_EmptyMap(t *testing.T) {
	personalRatio := map[uint32]float64{}

	if isWhiteExperienceGain(1, personalRatio, 0.5) {
		t.Errorf("Expected false for empty personalRatio map")
	}
}

func TestNewDamageEntryModel(t *testing.T) {
	entry := NewDamageEntryModel(123, 5000)

	if entry.characterId != 123 {
		t.Errorf("Expected characterId 123, got %d", entry.characterId)
	}
	if entry.damage != 5000 {
		t.Errorf("Expected damage 5000, got %d", entry.damage)
	}
}

func TestDamageDistributionModel_Accessors(t *testing.T) {
	solo := map[uint32]uint32{1: 100}
	party := map[uint32]map[uint32]uint32{}
	pr := map[uint32]float64{1: 0.5}

	ddm := DamageDistributionModel{
		solo:                   solo,
		party:                  party,
		personalRatio:          pr,
		experiencePerDamage:    2.5,
		standardDeviationRatio: 0.3,
	}

	if ddm.Solo()[1] != 100 {
		t.Errorf("Expected solo[1] = 100, got %d", ddm.Solo()[1])
	}
	if ddm.ExperiencePerDamage() != 2.5 {
		t.Errorf("Expected experiencePerDamage 2.5, got %f", ddm.ExperiencePerDamage())
	}
	if ddm.PersonalRatio()[1] != 0.5 {
		t.Errorf("Expected personalRatio[1] = 0.5, got %f", ddm.PersonalRatio()[1])
	}
	if ddm.StandardDeviationRatio() != 0.3 {
		t.Errorf("Expected standardDeviationRatio 0.3, got %f", ddm.StandardDeviationRatio())
	}
}
