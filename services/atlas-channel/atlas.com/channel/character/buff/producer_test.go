package buff

import (
	"bytes"
	"encoding/json"
	"testing"

	buffmsg "atlas-channel/kafka/message/buff"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
)

func TestCancelByTypesCommandProvider(t *testing.T) {
	f := field.NewBuilder(0, 0, 100000000).Build()
	types := []string{"STUN", "POISON"}

	msgs, err := CancelByTypesCommandProvider(f, 42, types)()
	if err != nil {
		t.Fatalf("provider returned error: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}

	var cmd buffmsg.Command[buffmsg.CancelByTypesCommandBody]
	if err := json.Unmarshal(msgs[0].Value, &cmd); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if cmd.Type != buffmsg.CommandTypeCancelByTypes {
		t.Errorf("Type = %q, want %q", cmd.Type, buffmsg.CommandTypeCancelByTypes)
	}
	if cmd.CharacterId != 42 {
		t.Errorf("CharacterId = %d, want 42", cmd.CharacterId)
	}
	if len(cmd.Body.Types) != 2 || cmd.Body.Types[0] != "STUN" || cmd.Body.Types[1] != "POISON" {
		t.Errorf("Body.Types = %v, want [STUN POISON]", cmd.Body.Types)
	}
}

// TestExpireCommandProvider guards CommandTypeExpire's wire value and the
// EXPIRE envelope shape the same way TestCancelByTypesCommandProvider guards
// CANCEL_BY_TYPES: a typo'd/renamed CommandTypeExpire would compile and run
// cleanly, but atlas-buffs would never claim the message and the CANCEL_DEBUFF
// sweep would silently do nothing (task-190 FR-2.6.1).
func TestExpireCommandProvider(t *testing.T) {
	instance := uuid.New()
	f := field.NewBuilder(world.Id(1), channel.Id(2), _map.Id(100000000)).SetInstance(instance).Build()

	msgs, err := ExpireCommandProvider(f, 42)()
	if err != nil {
		t.Fatalf("provider returned error: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}

	wantKey := producer.CreateKey(42)
	if !bytes.Equal(msgs[0].Key, wantKey) {
		t.Errorf("Key = %v, want %v (same derivation as sibling commands)", msgs[0].Key, wantKey)
	}

	// Assert the literal wire string, not the constant — comparing the
	// constant to itself would not catch an accidental rename of
	// CommandTypeExpire.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(msgs[0].Value, &raw); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	var typ string
	if err := json.Unmarshal(raw["type"], &typ); err != nil {
		t.Fatalf("unmarshal type failed: %v", err)
	}
	if typ != "EXPIRE" {
		t.Errorf("Type = %q, want %q", typ, "EXPIRE")
	}
	if body, ok := raw["body"]; !ok || string(body) != "{}" {
		t.Errorf("Body = %s, want {}", body)
	}

	var cmd buffmsg.Command[buffmsg.ExpireCommandBody]
	if err := json.Unmarshal(msgs[0].Value, &cmd); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if cmd.WorldId != world.Id(1) {
		t.Errorf("WorldId = %d, want 1", cmd.WorldId)
	}
	if cmd.ChannelId != channel.Id(2) {
		t.Errorf("ChannelId = %d, want 2", cmd.ChannelId)
	}
	if cmd.MapId != _map.Id(100000000) {
		t.Errorf("MapId = %d, want 100000000", cmd.MapId)
	}
	if cmd.Instance != instance {
		t.Errorf("Instance = %v, want %v", cmd.Instance, instance)
	}
	if cmd.CharacterId != 42 {
		t.Errorf("CharacterId = %d, want 42", cmd.CharacterId)
	}
}

// TestUpdateStatValueCommandProviderCarriesUpsertFields proves the channel-side
// provider puts the two fields task-216 added — createIfMissing and level — on
// the wire with the expected json tags and values, surviving a marshal/unmarshal
// round trip through this package's own Command[UpdateStatValueCommandBody].
// It does NOT enforce cross-module parity with atlas-buffs' owning declaration
// (services/atlas-buffs/atlas.com/buffs/kafka/message/character/kafka.go) — the
// two structs live in separate Go modules and are kept in sync by hand; a field
// name or json tag drifting between them fails no build and decodes into a zero
// value at runtime undetected by this test.
func TestUpdateStatValueCommandProviderCarriesUpsertFields(t *testing.T) {
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(100000000)).Build()

	msgs, err := UpdateStatValueCommandProvider(f, 1000, StatValueUpdate{
		SourceId: 5110001, StatType: "ENERGY_CHARGE", Operation: buffmsg.StatOperationIncrement,
		Amount: 204, Cap: 10000, CreateIfMissing: true, Level: 20,
	})()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}

	var cmd buffmsg.Command[buffmsg.UpdateStatValueCommandBody]
	if err := json.Unmarshal(msgs[0].Value, &cmd); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if cmd.Type != buffmsg.CommandTypeUpdateStatValue {
		t.Fatalf("type: got %q want %q", cmd.Type, buffmsg.CommandTypeUpdateStatValue)
	}
	if !cmd.Body.CreateIfMissing {
		t.Fatal("createIfMissing must survive the round trip")
	}
	if cmd.Body.Level != 20 {
		t.Fatalf("level: got %d want 20", cmd.Body.Level)
	}
	if cmd.Body.Amount != 204 || cmd.Body.Cap != 10000 {
		t.Fatalf("amount/cap: got %d/%d want 204/10000", cmd.Body.Amount, cmd.Body.Cap)
	}
}
