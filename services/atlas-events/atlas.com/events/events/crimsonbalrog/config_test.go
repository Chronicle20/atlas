package crimsonbalrog

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

const validConfig = `{
  "applicableRouteIds": ["boat-ellinia-orbis", "boat-orbis-ellinia"],
  "triggerDelay": "3m",
  "triggerDelayJitter": "60s",
  "attackProbability": 0.42,
  "monsterId": 8150000,
  "monsterCount": 2,
  "attackMaps": [{"mapId": 200090010, "spawnPositions": [{"x": 0, "y": 0}, {"x": 100, "y": 0}]}],
  "relatedMapIds": [200090011],
  "backgroundMusic": "Bgm04/ArabPirate",
  "visual": {"name": "CONTI_MOVE", "showState": 10, "showSubState": 4, "hideState": 10, "hideSubState": 5}
}`

func TestValidateAcceptsAWellFormedConfiguration(t *testing.T) {
	if err := NewHandler().ValidateConfiguration(json.RawMessage(validConfig)); err != nil {
		t.Fatalf("ValidateConfiguration: %v", err)
	}
}

// mutateConfig decodes validConfig into a generic map, applies mutate against
// the named test case, and re-marshals it. mutate is either a
// `"field": value` replacement snippet or a bare field name to delete.
func mutateConfig(t *testing.T, base string, name string) []byte {
	t.Helper()

	var m map[string]json.RawMessage
	if err := json.Unmarshal([]byte(base), &m); err != nil {
		t.Fatalf("unmarshal base config: %v", err)
	}

	switch name {
	case "zero monster count":
		m["monsterCount"] = json.RawMessage(`0`)
	case "probability above one":
		m["attackProbability"] = json.RawMessage(`1.5`)
	case "no attack maps":
		m["attackMaps"] = json.RawMessage(`[]`)
	case "fewer spawn positions than monsters":
		m["attackMaps"] = json.RawMessage(`[{"mapId": 200090010, "spawnPositions": [{"x": 0, "y": 0}]}]`)
	case "empty route id":
		m["applicableRouteIds"] = json.RawMessage(`["boat-ellinia-orbis", ""]`)
	default:
		t.Fatalf("mutateConfig: unknown case %q", name)
	}

	out, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal mutated config: %v", err)
	}
	return out
}

// FR-D6: the rejection must name the offending field, so an administrator sees
// what is wrong rather than "invalid configuration".
func TestValidateRejectsFieldByField(t *testing.T) {
	for _, tc := range []struct{ name, mutate, wantField string }{
		{"zero monster count", `"monsterCount": 2`, "monsterCount"},
		{"probability above one", `"attackProbability": 0.42`, "attackProbability"},
		{"no attack maps", `"attackMaps"`, "attackMaps"},
		{"empty route id", `"applicableRouteIds": ["boat-ellinia-orbis", "boat-orbis-ellinia"]`, "applicableRouteIds"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bad := mutateConfig(t, validConfig, tc.name)
			err := NewHandler().ValidateConfiguration(json.RawMessage(bad))
			if err == nil {
				t.Fatalf("expected rejection")
			}
			if !strings.Contains(err.Error(), tc.wantField) {
				t.Fatalf("error %q does not name %q", err, tc.wantField)
			}
		})
	}
}

// FR-B1: there are more spawn positions than monsters, or fewer. Fewer is
// rejected at write time rather than producing an out-of-range panic at spawn.
func TestValidateRejectsFewerSpawnPositionsThanMonsters(t *testing.T) {
	bad := mutateConfig(t, validConfig, "fewer spawn positions than monsters")
	err := NewHandler().ValidateConfiguration(json.RawMessage(bad))
	if err == nil {
		t.Fatalf("expected rejection")
	}
	if !strings.Contains(err.Error(), "spawnPositions") {
		t.Fatalf("error %q does not name %q", err, "spawnPositions")
	}
}

// workContextJSON marshals a WorkContext with the given voyage id (as a
// deterministic uuid derived from the string), worldId and channelId.
func workContextJSON(t *testing.T, voyageSeed string, worldId byte, channelId byte) json.RawMessage {
	t.Helper()

	wc := WorkContext{
		VoyageId:  uuid.NewSHA1(uuid.Nil, []byte(voyageSeed)),
		RouteId:   uuid.NewSHA1(uuid.Nil, []byte("route")),
		WorldId:   world.Id(worldId),
		ChannelId: channel.Id(channelId),
	}
	out, err := json.Marshal(wc)
	if err != nil {
		t.Fatalf("marshal work context: %v", err)
	}
	return out
}

// design §15.5: the key makes the occurrence one-per-voyage-per-channel, so two
// simultaneous voyages in different channels are independent (FR-N11) but one
// voyage cannot be attacked twice.
func TestConcurrencyKeyIsPerVoyageAndChannel(t *testing.T) {
	h := NewHandler()
	ctx := context.Background()

	a, err := h.ConcurrencyKey(ctx, workContextJSON(t, "voyage-1", 1, 4))
	if err != nil {
		t.Fatalf("ConcurrencyKey: %v", err)
	}
	same, _ := h.ConcurrencyKey(ctx, workContextJSON(t, "voyage-1", 1, 4))
	otherChannel, _ := h.ConcurrencyKey(ctx, workContextJSON(t, "voyage-1", 1, 5))
	otherVoyage, _ := h.ConcurrencyKey(ctx, workContextJSON(t, "voyage-2", 1, 4))

	if a != same {
		t.Fatalf("key not stable: %q vs %q", a, same)
	}
	if a == otherChannel || a == otherVoyage {
		t.Fatalf("key not discriminating: %q / %q / %q", a, otherChannel, otherVoyage)
	}
}
