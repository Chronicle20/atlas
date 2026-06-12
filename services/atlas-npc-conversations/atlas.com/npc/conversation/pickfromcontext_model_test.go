package conversation

import "testing"

func TestPickFromContextBuilder(t *testing.T) {
	m, err := NewPickFromContextBuilder().
		SetTitle("Which pet?").
		SetValuesContextKey("evolvablePets").
		SetLabelsContextKey("evolvablePetLabels").
		SetContextKey("selectedPetId").
		SetNextState("confirm").
		SetEmptyNextState("noEligible").
		Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if m.ValuesContextKey() != "evolvablePets" || m.LabelsContextKey() != "evolvablePetLabels" ||
		m.ContextKey() != "selectedPetId" || m.NextState() != "confirm" || m.EmptyNextState() != "noEligible" {
		t.Errorf("fields not set: %+v", m)
	}
}

func TestPickFromContextBuilderRequiresFields(t *testing.T) {
	if _, err := NewPickFromContextBuilder().SetNextState("x").SetEmptyNextState("y").Build(); err == nil {
		t.Error("expected error when valuesContextKey missing")
	}
	if _, err := NewPickFromContextBuilder().SetValuesContextKey("v").SetEmptyNextState("y").Build(); err == nil {
		t.Error("expected error when nextState missing")
	}
	if _, err := NewPickFromContextBuilder().SetValuesContextKey("v").SetNextState("x").Build(); err == nil {
		t.Error("expected error when emptyNextState missing")
	}
}

func TestStateBuilderSetPickFromContext(t *testing.T) {
	pfc, _ := NewPickFromContextBuilder().
		SetValuesContextKey("evolvablePets").SetNextState("confirm").SetEmptyNextState("noEligible").Build()
	s, err := NewStateBuilder().SetId("pick").SetPickFromContext(pfc).Build()
	if err != nil {
		t.Fatalf("state Build: %v", err)
	}
	if s.Type() != PickFromContextType {
		t.Errorf("Type() = %q, want %q", s.Type(), PickFromContextType)
	}
	if s.PickFromContext() == nil || s.PickFromContext().NextState() != "confirm" {
		t.Errorf("PickFromContext() not wired: %+v", s.PickFromContext())
	}
}
