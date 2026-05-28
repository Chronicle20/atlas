package conversation

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type expectedTarget struct {
	Stat  string
	Floor int
}

type scriptCase struct {
	name         string
	relPath      string
	advanceState string // the state whose operations array must contain rebalance_ap + change_job
	targets      []expectedTarget
	bannedStats  []string // stats that must NOT appear in any condition "type" or referenceId
}

// TestFirstJobScriptsUseRebalanceAP asserts that every first-job advancement
// script has rebalance_ap present immediately before change_job in the
// advancement state, that reset_stats is not used in the advancement state,
// and that no stat-minimum gate condition remains anywhere in the script.
func TestFirstJobScriptsUseRebalanceAP(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..", "..", "deploy", "seed", "gms", "83_1", "npc-conversations")

	cases := []scriptCase{
		{name: "Bowman", relPath: filepath.Join("npc", "npc-1012100.json"), advanceState: "firstJobAdvance", targets: []expectedTarget{{"dexterity", 25}}, bannedStats: []string{"dexterity"}},
		{name: "Warrior", relPath: filepath.Join("npc", "npc-1022000.json"), advanceState: "firstJobAdvance", targets: []expectedTarget{{"strength", 35}}, bannedStats: []string{"strength"}},
		{name: "Magician", relPath: filepath.Join("npc", "npc-1032001.json"), advanceState: "performFirstJobAdvancement", targets: []expectedTarget{{"intelligence", 20}}, bannedStats: []string{"intelligence"}},
		{name: "Thief", relPath: filepath.Join("npc", "npc-1052001.json"), advanceState: "firstJobAdvance", targets: []expectedTarget{{"dexterity", 25}}, bannedStats: []string{"dexterity"}},
		{name: "Pirate", relPath: filepath.Join("npc", "npc-1090000.json"), advanceState: "firstJobPerformAdvance", targets: []expectedTarget{{"dexterity", 20}}, bannedStats: []string{"dexterity"}},
		{name: "Dawn Warrior", relPath: filepath.Join("quests", "quest-20101.json"), advanceState: "performJobChange", targets: []expectedTarget{{"strength", 35}}, bannedStats: []string{"str"}},
		{name: "Blaze Wizard", relPath: filepath.Join("quests", "quest-20102.json"), advanceState: "performJobChange", targets: []expectedTarget{{"intelligence", 20}}, bannedStats: []string{"int"}},
		{name: "Wind Archer", relPath: filepath.Join("quests", "quest-20103.json"), advanceState: "performJobChange", targets: []expectedTarget{{"dexterity", 25}}, bannedStats: []string{"dex"}},
		{name: "Night Walker", relPath: filepath.Join("quests", "quest-20104.json"), advanceState: "performJobChange", targets: []expectedTarget{{"luck", 25}}, bannedStats: []string{"luk"}},
		{name: "Thunder Breaker", relPath: filepath.Join("quests", "quest-20105.json"), advanceState: "performJobChange", targets: []expectedTarget{{"strength", 20}, {"dexterity", 20}}, bannedStats: []string{"str", "dex"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join(root, tc.relPath))
			if err != nil {
				t.Fatalf("read %s: %v", tc.relPath, err)
			}
			var doc map[string]any
			if err := json.Unmarshal(raw, &doc); err != nil {
				t.Fatalf("parse %s: %v", tc.relPath, err)
			}

			asString := string(raw)
			for _, banned := range tc.bannedStats {
				banPatterns := []string{
					`"type": "` + banned + `"`,
					`"referenceId": "` + banned + `"`,
				}
				for _, pat := range banPatterns {
					if strings.Contains(asString, pat) {
						t.Errorf("%s: forbidden stat gate pattern remains: %q", tc.relPath, pat)
					}
				}
			}

			if strings.Contains(asString, `"reset_stats"`) && opsOfStateContain(t, doc, tc.advanceState, "reset_stats") {
				t.Errorf("%s: reset_stats must not appear in %q advancement operations", tc.relPath, tc.advanceState)
			}

			ops := collectOps(t, doc, tc.advanceState)
			rebalanceIdx := -1
			changeJobIdx := -1
			for i, op := range ops {
				typ := stringField(op, "type")
				if typ == "rebalance_ap" && rebalanceIdx < 0 {
					rebalanceIdx = i
				}
				if typ == "change_job" && changeJobIdx < 0 {
					changeJobIdx = i
				}
			}
			if rebalanceIdx < 0 {
				t.Fatalf("%s: no rebalance_ap in %q operations", tc.relPath, tc.advanceState)
			}
			if changeJobIdx < 0 {
				t.Fatalf("%s: no change_job in %q operations", tc.relPath, tc.advanceState)
			}
			if rebalanceIdx >= changeJobIdx {
				t.Errorf("%s: rebalance_ap (at %d) must precede change_job (at %d)", tc.relPath, rebalanceIdx, changeJobIdx)
			}

			rebalance := ops[rebalanceIdx]
			params, _ := rebalance["params"].(map[string]any)
			targetsStr, _ := params["targets"].(string)
			var gotTargets []map[string]any
			if err := json.Unmarshal([]byte(targetsStr), &gotTargets); err != nil {
				t.Fatalf("%s: cannot parse targets JSON %q: %v", tc.relPath, targetsStr, err)
			}
			if len(gotTargets) != len(tc.targets) {
				t.Fatalf("%s: expected %d targets, got %d", tc.relPath, len(tc.targets), len(gotTargets))
			}
			for i, want := range tc.targets {
				gotStat, _ := gotTargets[i]["stat"].(string)
				gotFloor := toInt(gotTargets[i]["floor"])
				if gotStat != want.Stat || gotFloor != want.Floor {
					t.Errorf("%s: target[%d]: got {%s,%d}, want {%s,%d}",
						tc.relPath, i, gotStat, gotFloor, want.Stat, want.Floor)
				}
			}
		})
	}
}

// helpers

// unwrapDoc unwraps the seed catalog envelope {"data": {"attributes": {...}}}
// if present, returning the attributes map; otherwise returns doc unchanged.
func unwrapDoc(doc map[string]any) map[string]any {
	if data, ok := doc["data"].(map[string]any); ok {
		if attrs, ok := data["attributes"].(map[string]any); ok {
			return attrs
		}
	}
	return doc
}

func collectOps(t *testing.T, doc map[string]any, stateId string) []map[string]any {
	t.Helper()
	inner := unwrapDoc(doc)
	if states, ok := inner["states"].([]any); ok {
		if ops := opsFromStates(states, stateId); ops != nil {
			return ops
		}
	}
	for _, key := range []string{"startStateMachine", "endStateMachine"} {
		if sm, ok := inner[key].(map[string]any); ok {
			if states, ok := sm["states"].([]any); ok {
				if ops := opsFromStates(states, stateId); ops != nil {
					return ops
				}
			}
		}
	}
	t.Fatalf("no state %q found", stateId)
	return nil
}

func opsFromStates(states []any, stateId string) []map[string]any {
	for _, s := range states {
		sm, ok := s.(map[string]any)
		if !ok {
			continue
		}
		id, _ := sm["id"].(string)
		if id != stateId {
			continue
		}
		ga, _ := sm["genericAction"].(map[string]any)
		if ga == nil {
			return nil
		}
		opsRaw, _ := ga["operations"].([]any)
		out := make([]map[string]any, 0, len(opsRaw))
		for _, op := range opsRaw {
			if m, ok := op.(map[string]any); ok {
				out = append(out, m)
			}
		}
		return out
	}
	return nil
}

func opsOfStateContain(t *testing.T, doc map[string]any, stateId, opType string) bool {
	t.Helper()
	for _, op := range collectOps(t, doc, stateId) {
		if stringField(op, "type") == opType {
			return true
		}
	}
	return false
}

func stringField(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return v
}

func toInt(v any) int {
	switch x := v.(type) {
	case float64:
		return int(x)
	case int:
		return x
	case string:
		var i int
		_ = json.Unmarshal([]byte(x), &i)
		return i
	}
	return 0
}
