package _map

import (
	"encoding/json"
	"sort"
	"testing"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
)

// TestDestroyFieldCommandProvider_MatchesConsumerEnvelope pins the exact JSON
// key set emitted by destroyFieldCommandProvider against the contract
// consumed by atlas-monsters' fieldCommand
// (services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/kafka.go).
// The consumer's body is struct{}, so a silent field-name drift here fails
// open with no error anywhere else in the system; verify.sh cannot see it.
func TestDestroyFieldCommandProvider_MatchesConsumerEnvelope(t *testing.T) {
	instance := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	f := field.NewBuilder(world.Id(1), channel.Id(2), _map.Id(920010920)).SetInstance(instance).Build()

	msgs, err := destroyFieldCommandProvider(f)()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected exactly 1 message, got %d", len(msgs))
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(msgs[0].Value, &raw); err != nil {
		t.Fatalf("failed to unmarshal message value: %v", err)
	}

	expectedKeys := []string{"worldId", "channelId", "mapId", "instance", "type", "body"}
	if len(raw) != len(expectedKeys) {
		actualKeys := make([]string, 0, len(raw))
		for k := range raw {
			actualKeys = append(actualKeys, k)
		}
		sort.Strings(actualKeys)
		t.Fatalf("expected exactly %d keys %v, got %d keys %v", len(expectedKeys), expectedKeys, len(raw), actualKeys)
	}
	for _, k := range expectedKeys {
		if _, ok := raw[k]; !ok {
			actualKeys := make([]string, 0, len(raw))
			for k := range raw {
				actualKeys = append(actualKeys, k)
			}
			sort.Strings(actualKeys)
			t.Fatalf("missing expected key %q, got keys %v", k, actualKeys)
		}
	}

	assertJSONField(t, raw, "worldId", "1")
	assertJSONField(t, raw, "channelId", "2")
	assertJSONField(t, raw, "mapId", "920010920")
	assertJSONField(t, raw, "instance", `"11111111-2222-3333-4444-555555555555"`)
	assertJSONField(t, raw, "type", `"DESTROY_FIELD"`)
	assertJSONField(t, raw, "body", "{}")

	expectedKey := producer.CreateKey(int(f.MapId()))
	if string(msgs[0].Key) != string(expectedKey) {
		t.Fatalf("expected key %v, got %v", expectedKey, msgs[0].Key)
	}
}

func assertJSONField(t *testing.T, raw map[string]json.RawMessage, key string, expected string) {
	t.Helper()
	actual, ok := raw[key]
	if !ok {
		t.Fatalf("missing key %q", key)
	}
	if string(actual) != expected {
		t.Fatalf("expected %q to be %s, got %s", key, expected, string(actual))
	}
}
