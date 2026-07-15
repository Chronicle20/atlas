package drop

import (
	"math"
	"testing"
)

func TestGetRandomStat_ZeroStaysZero(t *testing.T) {
	for i := 0; i < 100; i++ {
		if got := getRandomStat(0, 5); got != 0 {
			t.Fatalf("zero stat must stay zero, got %d", got)
		}
	}
}

func TestGetRandomStat_Bounds(t *testing.T) {
	// maxRange = min(ceil(default*0.1), max); result in [default-maxRange, default+maxRange]
	for i := 0; i < 1000; i++ {
		def, maxSpread := uint16(100), uint16(5)
		maxRange := math.Min(math.Ceil(float64(def)*0.1), float64(maxSpread)) // = 5
		got := getRandomStat(def, maxSpread)
		if float64(got) < float64(def)-maxRange || float64(got) > float64(def)+maxRange {
			t.Fatalf("stat %d outside [%f,%f]", got, float64(def)-maxRange, float64(def)+maxRange)
		}
	}
}

func TestIsEquipment(t *testing.T) {
	if !isEquipment(1302000) {
		t.Fatal("1302000 is equipment")
	}
	if isEquipment(2000000) {
		t.Fatal("2000000 is not equipment")
	}
}
