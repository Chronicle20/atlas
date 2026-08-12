package monster

import (
	monster2 "atlas-channel/kafka/message/monster"
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

func magnetTestField() field.Model {
	return field.NewBuilder(1, 2, 100000000).SetInstance(uuid.Nil).Build()
}

// TestClearAggroCommandProviderShape pins the envelope and the deliberately
// empty body (FR-4.3). Every handler on COMMAND_TOPIC_MONSTER unmarshals every
// message into its own body type, so an empty body cannot collide with a
// sibling's field types.
func TestClearAggroCommandProviderShape(t *testing.T) {
	msgs, err := ClearAggroCommandProvider(magnetTestField(), 4242)()
	if err != nil {
		t.Fatalf("provider returned error: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("provider produced %d messages, want 1", len(msgs))
	}

	var c monster2.Command[monster2.ClearAggroCommandBody]
	if err := json.Unmarshal(msgs[0].Value, &c); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if c.Type != monster2.CommandTypeClearAggro {
		t.Fatalf("Type = %q, want %q", c.Type, monster2.CommandTypeClearAggro)
	}
	if c.MonsterId != 4242 {
		t.Fatalf("MonsterId = %d, want 4242", c.MonsterId)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(msgs[0].Value, &raw); err != nil {
		t.Fatalf("unmarshal raw: %v", err)
	}
	if string(raw["body"]) != "{}" {
		t.Fatalf("body = %s, want {} — the clear-aggro body must carry no fields", raw["body"])
	}
}

func TestForceControlCommandProviderShape(t *testing.T) {
	msgs, err := ForceControlCommandProvider(magnetTestField(), 4242, 777)()
	if err != nil {
		t.Fatalf("provider returned error: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("provider produced %d messages, want 1", len(msgs))
	}

	var c monster2.Command[monster2.ForceControlCommandBody]
	if err := json.Unmarshal(msgs[0].Value, &c); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if c.Type != monster2.CommandTypeForceControl {
		t.Fatalf("Type = %q, want %q", c.Type, monster2.CommandTypeForceControl)
	}
	if c.MonsterId != 4242 {
		t.Fatalf("MonsterId = %d, want 4242", c.MonsterId)
	}
	if c.Body.CharacterId != 777 {
		t.Fatalf("Body.CharacterId = %d, want 777", c.Body.CharacterId)
	}
}

// TestMonsterCommandsShareMonsterKey pins the ordering contract: both commands
// key on the monster id, so CLEAR_AGGRO then FORCE_CONTROL for the same monster
// land on the same partition in emit order. Reversing them would have the wipe
// immediately clear the aggro flag the handover just set.
func TestMonsterCommandsShareMonsterKey(t *testing.T) {
	clear, err := ClearAggroCommandProvider(magnetTestField(), 4242)()
	if err != nil {
		t.Fatalf("clear provider returned error: %v", err)
	}
	force, err := ForceControlCommandProvider(magnetTestField(), 4242, 777)()
	if err != nil {
		t.Fatalf("force provider returned error: %v", err)
	}
	if string(clear[0].Key) != string(force[0].Key) {
		t.Fatalf("keys differ (%x vs %x); both commands must key on the monster id",
			clear[0].Key, force[0].Key)
	}
}
