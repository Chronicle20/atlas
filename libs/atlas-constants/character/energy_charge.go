package character

// The two magnitudes the ENERGY_CHARGE temporary stat
// (TemporaryStatTypeEnergyCharge) can carry that mean something beyond "the bar
// is this full". They are shared rather than redeclared per service because
// three separate call sites across two Go modules must agree on them —
// atlas-channel accumulates up to the cap and promotes at it, atlas-channel's
// buff consumer recognizes the promotion, and atlas-effective-stats grants the
// weapon-attack bonus only while the sentinel is set. A copy that drifts from
// the others fails no build; it silently desyncs the accumulation ceiling from
// the promotion and the stat payoff.
const (
	// EnergyChargeCap is the accumulation ceiling. Reaching it exactly promotes
	// the character to the charged state; the bar is clamped here, never above.
	EnergyChargeCap = int32(10000)

	// EnergyChargedValue is the charged-state SENTINEL, not a bar reading. It
	// is deliberately outside the 0..EnergyChargeCap accumulation range so the
	// "already charged, stop accumulating" test is a single equality check.
	EnergyChargedValue = int32(15000)
)
