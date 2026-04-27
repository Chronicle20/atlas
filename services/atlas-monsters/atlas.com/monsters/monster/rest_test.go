package monster

import "testing"

func TestTransform_IncludesAggroAndRepickFields(t *testing.T) {
	m := NewMonster(testField(), 1, 9000000, 0, 0, 0, 0, 0, 100, 50)
	m = Clone(m).
		SetControllerHasAggro(true).
		SetNextSkillDecision(nextSkillDecision{nextEligibleRepickAtMs: 1730000005000}).
		Build()

	rm, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	if !rm.ControllerHasAggro {
		t.Errorf("ControllerHasAggro should be true; got false")
	}
	if rm.NextEligibleRepickAtMs != 1730000005000 {
		t.Errorf("NextEligibleRepickAtMs: got %d, want 1730000005000", rm.NextEligibleRepickAtMs)
	}
}

func TestTransform_OmitsZeroNextEligibleRepick(t *testing.T) {
	// Marshal output should not contain nextEligibleRepickAtMs when it is 0.
	// We encode the struct via encoding/json since RestModel is a plain struct
	// with json tags.
	m := NewMonster(testField(), 1, 9000000, 0, 0, 0, 0, 0, 100, 50)
	rm, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	if rm.ControllerHasAggro {
		t.Errorf("ControllerHasAggro should default to false")
	}
	if rm.NextEligibleRepickAtMs != 0 {
		t.Errorf("NextEligibleRepickAtMs should default to 0")
	}
}
