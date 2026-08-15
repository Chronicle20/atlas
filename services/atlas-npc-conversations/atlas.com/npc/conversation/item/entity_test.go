package item

import (
	"testing"

	"github.com/google/uuid"
)

func TestEntityRoundTrip(t *testing.T) {
	state := buildFixtureState(t, "intro")
	in, err := NewBuilder().
		SetItemId(2430008).
		SetNpcId(2084002).
		SetScriptName("compassUse").
		SetStartState("intro").
		AddState(state).
		Build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	tenantId := uuid.New()
	e, err := ToEntity(in, tenantId)
	if err != nil {
		t.Fatalf("ToEntity: %v", err)
	}
	if e.TableName() != "item_conversations" {
		t.Errorf("table: got %q", e.TableName())
	}
	if e.ItemID != 2430008 || e.NpcID != 2084002 || e.ScriptName != "compassUse" {
		t.Errorf("columns: item %d npc %d script %q", e.ItemID, e.NpcID, e.ScriptName)
	}
	if e.TenantID != tenantId {
		t.Errorf("tenant: got %s want %s", e.TenantID, tenantId)
	}

	out, err := Make(e)
	if err != nil {
		t.Fatalf("Make: %v", err)
	}
	if out.ItemId() != in.ItemId() || out.NpcId() != in.NpcId() || out.ScriptName() != in.ScriptName() {
		t.Errorf("round-trip mismatch: %+v", out)
	}
	if out.StartState() != "intro" || len(out.States()) != 1 {
		t.Errorf("state machine lost in round-trip: start %q states %d", out.StartState(), len(out.States()))
	}
}
