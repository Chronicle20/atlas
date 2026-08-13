package configuration

import (
	"fmt"
	"math"
)

// ValidateTiers enforces FR-9.3: strictly descending thresholds and rates in
// [0, 1]. An invalid table is a loud error, never a silent partial accept.
func ValidateTiers(tiers []Tier) error {
	for i, t := range tiers {
		if t.Rate < 0 || t.Rate > 1 {
			return fmt.Errorf("trade tax tier %d: rate %.4f outside [0, 1]", i, t.Rate)
		}
		if i > 0 && t.Threshold >= tiers[i-1].Threshold {
			return fmt.Errorf("trade tax tier %d: threshold %d is not strictly below the previous tier's %d", i, t.Threshold, tiers[i-1].Threshold)
		}
	}
	return nil
}

// Tax computes the meso deduction for one side's staged amount:
// delivered = m - floor(m * rate(m)). The difference is DESTROYED — the giver's
// negative award_mesos is the full m, the receiver's positive award is
// delivered, and no third party is credited (design §6.5).
//
// Arithmetic width: meso is uint32, and every uint32 is exactly representable
// in float64 (32 bits of magnitude against a 53-bit mantissa), so widening the
// amount loses nothing. A Model's rates are validated into [0, 1], which bounds
// the product by the amount itself — the multiply cannot overflow, math.Floor
// truncates it to an integral float below 2^32, and the uint32 conversion is
// therefore lossless. delivered = amount - tax likewise cannot underflow.
func Tax(m Model, amount uint32) (tax uint32, delivered uint32) {
	if !m.TaxEnabled() || amount == 0 {
		return 0, amount
	}
	for _, tier := range m.taxTiers {
		if amount >= tier.Threshold {
			tax = uint32(math.Floor(float64(amount) * tier.Rate))
			return tax, amount - tax
		}
	}
	return 0, amount
}
