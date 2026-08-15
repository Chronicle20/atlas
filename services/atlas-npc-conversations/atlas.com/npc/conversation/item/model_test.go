package item

import (
	"atlas-npc-conversations/conversation"
	"testing"
)

// buildFixtureState builds a minimal valid conversation.StateModel for use in
// item package tests.
//
// The brief's original fixture assumed a `conversation.NewStateModelBuilder()`
// with a bare `.SetId(...).Build()`. That function does not exist; the real
// builder is `conversation.NewStateBuilder()` (verified via
// `grep -n "func NewStateModelBuilder\|StateModelBuilder) Set\|func (b \*StateModelBuilder) Build" conversation/model.go`,
// which returned zero matches), and its Build() requires a state type to be
// set — the switch in StateBuilder.Build() falls through to
// `errors.New("invalid state type")` when none of the Set*Action methods was
// called. A genericAction with a single (unconditional) outcome is the
// smallest state that satisfies both StateBuilder.Build() and
// GenericActionBuilder.Build()'s "at least one operation or outcome" check.
func buildFixtureState(t *testing.T, id string) conversation.StateModel {
	t.Helper()

	outcome, err := conversation.NewOutcomeBuilder().Build()
	if err != nil {
		t.Fatalf("building fixture outcome: %v", err)
	}
	ga, err := conversation.NewGenericActionBuilder().AddOutcome(outcome).Build()
	if err != nil {
		t.Fatalf("building fixture genericAction: %v", err)
	}
	state, err := conversation.NewStateBuilder().SetId(id).SetGenericAction(ga).Build()
	if err != nil {
		t.Fatalf("building fixture state: %v", err)
	}
	return state
}

func TestBuilderRequiresItemIdStartStateAndStates(t *testing.T) {
	state := buildFixtureState(t, "intro")

	if _, err := NewBuilder().SetStartState("intro").AddState(state).Build(); err == nil {
		t.Error("expected error when itemId is unset")
	}
	if _, err := NewBuilder().SetItemId(2430008).AddState(state).Build(); err == nil {
		t.Error("expected error when startState is unset")
	}
	if _, err := NewBuilder().SetItemId(2430008).SetStartState("intro").Build(); err == nil {
		t.Error("expected error when states is empty")
	}

	m, err := NewBuilder().
		SetItemId(2430008).
		SetNpcId(2084002).
		SetScriptName("compassUse").
		SetStartState("intro").
		AddState(state).
		Build()
	if err != nil {
		t.Fatalf("valid build: %v", err)
	}
	if m.ItemId() != 2430008 || m.NpcId() != 2084002 || m.ScriptName() != "compassUse" {
		t.Errorf("round-trip: got item %d npc %d script %q", m.ItemId(), m.NpcId(), m.ScriptName())
	}
}

// FindState is the conversation.StateContainer contract — it is what lets the
// existing ProcessState/Continue/End machinery drive an item conversation with
// no changes.
func TestModelImplementsStateContainer(t *testing.T) {
	var _ conversation.StateContainer = Model{}

	state := buildFixtureState(t, "intro")
	m, err := NewBuilder().SetItemId(2430013).SetStartState("intro").AddState(state).Build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if got, err := m.FindState("intro"); err != nil || got.Id() != "intro" {
		t.Errorf("FindState(intro): got %q err %v", got.Id(), err)
	}
	if _, err := m.FindState("nope"); err == nil {
		t.Error("FindState on an unknown state must error")
	}
}
