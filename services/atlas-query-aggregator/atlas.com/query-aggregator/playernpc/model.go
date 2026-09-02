package playernpc

// EligibilityModel is the domain view of the eligibility predicate.
type EligibilityModel struct {
	eligible bool
	reason   string
}

func (m EligibilityModel) Eligible() bool { return m.eligible }
func (m EligibilityModel) Reason() string { return m.reason }

// NewEligibilityModel builds an EligibilityModel directly -- used by the
// GetEligibility implementation to wrap a decoded REST response, and by
// tests to build fakes without exporting the struct's fields.
func NewEligibilityModel(eligible bool, reason string) EligibilityModel {
	return EligibilityModel{eligible: eligible, reason: reason}
}

// NewUnavailableEligibility builds the fail-closed EligibilityModel a caller
// returns when the eligibility endpoint could not be reached or has no
// processor wired -- the graceful-degradation result GetPlayerNpcEligibility
// (validation/context.go) returns rather than propagating an error the
// evaluator contract has no channel for.
func NewUnavailableEligibility(reason string) EligibilityModel {
	return NewEligibilityModel(false, reason)
}
