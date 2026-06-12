package conversation

import "testing"

func TestPickFromContextRoundTrip(t *testing.T) {
	pfc, _ := NewPickFromContextBuilder().
		SetTitle("Which pet?").
		SetValuesContextKey("evolvablePets").
		SetLabelsContextKey("evolvablePetLabels").
		SetContextKey("selectedPetId").
		SetNextState("confirm").
		SetEmptyNextState("noEligible").
		Build()
	state, err := NewStateBuilder().SetId("pick").SetPickFromContext(pfc).Build()
	if err != nil {
		t.Fatalf("state build: %v", err)
	}

	rest, err := TransformState(state)
	if err != nil {
		t.Fatalf("TransformState: %v", err)
	}
	if rest.StateType != string(PickFromContextType) || rest.PickFromContext == nil {
		t.Fatalf("transform missing pickFromContext: %+v", rest)
	}
	if rest.PickFromContext.ValuesContextKey != "evolvablePets" || rest.PickFromContext.EmptyNextState != "noEligible" {
		t.Errorf("rest fields wrong: %+v", rest.PickFromContext)
	}

	back, err := ExtractState(rest)
	if err != nil {
		t.Fatalf("ExtractState: %v", err)
	}
	got := back.PickFromContext()
	if got == nil || got.ValuesContextKey() != "evolvablePets" || got.LabelsContextKey() != "evolvablePetLabels" ||
		got.ContextKey() != "selectedPetId" || got.NextState() != "confirm" || got.EmptyNextState() != "noEligible" {
		t.Errorf("round-trip mismatch: %+v", got)
	}
}
