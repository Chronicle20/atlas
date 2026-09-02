package main

import (
	"reflect"
	"strings"
	"testing"
)

func TestConvertBody(t *testing.T) {
	tests := []struct {
		name      string
		reactorId string
		hitBody   string
		actBody   string
		wantHit   []ruleDoc
		wantAct   []ruleDoc
	}{
		{
			name:      "bare drop",
			reactorId: "1002008",
			actBody:   `rm.dropItems();`,
			wantHit:   []ruleDoc{},
			wantAct: []ruleDoc{{
				Id:         "drop_items",
				Operations: []opDoc{{Type: "drop_items", Params: nil}},
			}},
		},
		{
			name:      "drop 4-arg",
			reactorId: "1012000",
			actBody:   `rm.dropItems(true, 2, 20, 40);`,
			wantHit:   []ruleDoc{},
			wantAct: []ruleDoc{{
				Id: "drop_items",
				Operations: []opDoc{{Type: "drop_items", Params: map[string]string{
					"meso": "true", "mesoChance": "2", "mesoMin": "20", "mesoMax": "40",
				}}},
			}},
		},
		{
			name:      "drop 5-arg",
			reactorId: "2001000",
			actBody:   `rm.dropItems(true, 2, 8, 15, 1);`,
			wantHit:   []ruleDoc{},
			wantAct: []ruleDoc{{
				Id: "drop_items",
				Operations: []opDoc{{Type: "drop_items", Params: map[string]string{
					"meso": "true", "mesoChance": "2", "mesoMin": "8", "mesoMax": "15", "minItems": "1",
				}}},
			}},
		},
		{
			name:      "drop 5-arg false",
			reactorId: "3102000",
			actBody:   `rm.dropItems(false, 0, 0, 0, 3);`,
			wantHit:   []ruleDoc{},
			wantAct: []ruleDoc{{
				Id: "drop_items",
				Operations: []opDoc{{Type: "drop_items", Params: map[string]string{
					"meso": "false", "mesoChance": "0", "mesoMin": "0", "mesoMax": "0", "minItems": "3",
				}}},
			}},
		},
		{
			name:      "bare spray, hit",
			reactorId: "2612004",
			hitBody:   `rm.sprayItems();`,
			wantHit: []ruleDoc{{
				Id:         "spray_items",
				Operations: []opDoc{{Type: "spray_items", Params: nil}},
			}},
			wantAct: []ruleDoc{},
		},
		{
			name:      "spray 5-arg",
			reactorId: "1052001",
			actBody:   `rm.sprayItems(true, 1, 500, 1000, 15);`,
			wantHit:   []ruleDoc{},
			wantAct: []ruleDoc{{
				Id: "spray_items",
				Operations: []opDoc{{Type: "spray_items", Params: map[string]string{
					"meso": "true", "mesoChance": "1", "mesoMin": "500", "mesoMax": "1000", "minItems": "15",
				}}},
			}},
		},
		{
			name:      "spawn 1-arg",
			reactorId: "1021000",
			actBody:   `rm.spawnMonster(9300091);`,
			wantHit:   []ruleDoc{},
			wantAct: []ruleDoc{{
				Id:         "spawn_monster",
				Operations: []opDoc{{Type: "spawn_monster", Params: map[string]string{"monsterId": "9300091"}}},
			}},
		},
		{
			name:      "spawn 2-arg",
			reactorId: "2201000",
			actBody:   `rm.spawnMonster(9300011, 10);`,
			wantHit:   []ruleDoc{},
			wantAct: []ruleDoc{{
				Id: "spawn_monster",
				Operations: []opDoc{{Type: "spawn_monster", Params: map[string]string{
					"monsterId": "9300011", "count": "10",
				}}},
			}},
		},
		{
			name:      "spawn 4-arg",
			reactorId: "8001000",
			actBody:   `rm.spawnMonster(9400112, 1, 420, 160);`,
			wantHit:   []ruleDoc{},
			wantAct: []ruleDoc{{
				Id: "spawn_monster",
				Operations: []opDoc{{Type: "spawn_monster", Params: map[string]string{
					"monsterId": "9400112", "count": "1", "x": "420", "y": "160",
				}}},
			}},
		},
		{
			name:      "spawn 4-arg negative x",
			reactorId: "9201000",
			actBody:   `rm.spawnMonster(9300033, 8, -100, 50);`,
			wantHit:   []ruleDoc{},
			wantAct: []ruleDoc{{
				Id: "spawn_monster",
				Operations: []opDoc{{Type: "spawn_monster", Params: map[string]string{
					"monsterId": "9300033", "count": "8", "x": "-100", "y": "50",
				}}},
			}},
		},
		{
			name:      "weaken, no semicolon",
			reactorId: "2119000",
			hitBody:   `rm.weakenAreaBoss(6090000, "As the tombstone lit up and vanished, Lich lost all his magic abilities.")`,
			wantHit: []ruleDoc{{
				Id: "weaken_area_boss",
				Operations: []opDoc{{Type: "weaken_area_boss", Params: map[string]string{
					"monsterId": "6090000",
					"message":   "As the tombstone lit up and vanished, Lich lost all his magic abilities.",
				}}},
			}},
			wantAct: []ruleDoc{},
		},
		{
			name:      "weaken, semicolon",
			reactorId: "2119004",
			actBody:   `rm.weakenAreaBoss(6090001, "The light at the altar appeases the hatred of the Snow Witch. The force of the Witch has weakened.");`,
			wantHit:   []ruleDoc{},
			wantAct: []ruleDoc{{
				Id: "weaken_area_boss",
				Operations: []opDoc{{Type: "weaken_area_boss", Params: map[string]string{
					"monsterId": "6090001",
					"message":   "The light at the altar appeases the hatred of the Snow Witch. The force of the Witch has weakened.",
				}}},
			}},
		},
		{
			name:      "guard hoist",
			reactorId: "2119000",
			hitBody: "if (rm.getReactor().getState() !== 0) {\n" +
				"    return\n" +
				"}\n" +
				`rm.weakenAreaBoss(6090000, "As the tombstone lit up and vanished, Lich lost all his magic abilities.")`,
			wantHit: []ruleDoc{{
				Id:         "weaken_area_boss",
				Conditions: []condDoc{{Type: "reactor_state", Operator: "=", Value: "0"}},
				Operations: []opDoc{{Type: "weaken_area_boss", Params: map[string]string{
					"monsterId": "6090000",
					"message":   "As the tombstone lit up and vanished, Lich lost all his magic abilities.",
				}}},
			}},
			wantAct: []ruleDoc{},
		},
		{
			name:      "loop unroll, one op",
			reactorId: "2201001",
			actBody: "for (var i = 0; i < 3; i++) {\n" +
				"    rm.spawnMonster(9300007);\n" +
				"}",
			wantHit: []ruleDoc{},
			wantAct: []ruleDoc{{
				Id: "spawn_monster",
				Operations: []opDoc{{Type: "spawn_monster", Params: map[string]string{
					"monsterId": "9300007", "count": "3",
				}}},
			}},
		},
		{
			name:      "loop unroll, two ops",
			reactorId: "2511001",
			actBody: "for (var i = 0; i < 6; i++) {\n" +
				"    rm.spawnMonster(9300124);\n" +
				"    rm.spawnMonster(9300125);\n" +
				"}",
			wantHit: []ruleDoc{},
			wantAct: []ruleDoc{{
				Id: "spawn_monster",
				Operations: []opDoc{
					{Type: "spawn_monster", Params: map[string]string{"monsterId": "9300124", "count": "6"}},
					{Type: "spawn_monster", Params: map[string]string{"monsterId": "9300125", "count": "6"}},
				},
			}},
		},
		{
			name:      "increment idiom",
			reactorId: "2511000",
			actBody: `var eim = rm.getPlayer().getEventInstance();
var now = eim.getIntProperty("openedBoxes");
var nextNum = now + 1;
eim.setIntProperty("openedBoxes", nextNum);
rm.spawnMonster(9300109, 3);
rm.spawnMonster(9300110, 5);`,
			wantHit: []ruleDoc{},
			wantAct: []ruleDoc{{
				Id: "update_pq_state_spawn_monster",
				Operations: []opDoc{
					{Type: "update_pq_state", Params: map[string]string{"increments": "openedBoxes"}},
					{Type: "spawn_monster", Params: map[string]string{"monsterId": "9300109", "count": "3"}},
					{Type: "spawn_monster", Params: map[string]string{"monsterId": "9300110", "count": "5"}},
				},
			}},
		},
		{
			name:      "increment + spray",
			reactorId: "2512001",
			actBody: `var eim = rm.getPlayer().getEventInstance();
var now = eim.getIntProperty("openedChests");
var nextNum = now + 1;
eim.setIntProperty("openedChests", nextNum);
rm.sprayItems(true, 1, 50, 100, 15);`,
			wantHit: []ruleDoc{},
			wantAct: []ruleDoc{{
				Id: "update_pq_state_spray_items",
				Operations: []opDoc{
					{Type: "update_pq_state", Params: map[string]string{"increments": "openedChests"}},
					{Type: "spray_items", Params: map[string]string{
						"meso": "true", "mesoChance": "1", "mesoMin": "50", "mesoMax": "100", "minItems": "15",
					}},
				},
			}},
		},
		{
			name:      "setProperty after drop",
			reactorId: "2002003",
			actBody: `rm.dropItems();
var eim = rm.getEventInstance();
eim.setProperty("statusStg7", "1");`,
			wantHit: []ruleDoc{},
			wantAct: []ruleDoc{{
				Id: "drop_items_update_pq_state",
				Operations: []opDoc{
					{Type: "drop_items", Params: nil},
					{Type: "update_pq_state", Params: map[string]string{"updates": "statusStg7=1"}},
				},
			}},
		},
		{
			name:      "inline setProperty",
			reactorId: "2008006",
			actBody:   `rm.getEventInstance().setProperty("statusStg3", "0");`,
			wantHit:   []ruleDoc{},
			wantAct: []ruleDoc{{
				Id:         "update_pq_state",
				Operations: []opDoc{{Type: "update_pq_state", Params: map[string]string{"updates": "statusStg3=0"}}},
			}},
		},
		{
			name:      "null-guard erased",
			reactorId: "9208009",
			actBody: "if (rm.getEventInstance() != null) {\n" +
				`    rm.getEventInstance().setProperty("canRevive", "1");` + "\n" +
				"}",
			wantHit: []ruleDoc{},
			wantAct: []ruleDoc{{
				Id:         "update_pq_state",
				Operations: []opDoc{{Type: "update_pq_state", Params: map[string]string{"updates": "canRevive=1"}}},
			}},
		},
		{
			name:      "empty act",
			reactorId: "9018000",
			wantHit:   []ruleDoc{},
			wantAct:   []ruleDoc{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := sourceScript{Id: tt.reactorId, HitBody: tt.hitBody, ActBody: tt.actBody}
			doc, err := convertScript(s)
			if err != nil {
				t.Fatalf("convertScript: %v", err)
			}
			if !reflect.DeepEqual(doc.HitRules, tt.wantHit) {
				t.Errorf("HitRules =\n%#v\nwant\n%#v", doc.HitRules, tt.wantHit)
			}
			if !reflect.DeepEqual(doc.ActRules, tt.wantAct) {
				t.Errorf("ActRules =\n%#v\nwant\n%#v", doc.ActRules, tt.wantAct)
			}
		})
	}
}

func TestConvertBody_NegativeCases(t *testing.T) {
	tests := []struct {
		name      string
		reactorId string
		body      string
		wantLine  string
	}{
		{
			name:      "unknown call",
			reactorId: "9999901",
			body:      `rm.doSomethingElse(1);`,
			wantLine:  `rm.doSomethingElse(1);`,
		},
		{
			name:      "random spawn",
			reactorId: "9999902",
			body:      `rm.spawnMonster(Math.random() >= .6 ? 9300049 : 9300048);`,
			wantLine:  `rm.spawnMonster(Math.random() >= .6 ? 9300049 : 9300048);`,
		},
		{
			name:      "non-unit increment",
			reactorId: "9999903",
			body: `var now = eim.getIntProperty("k");
var nextNum = now + 2;
eim.setIntProperty("k", nextNum);`,
			wantLine: `var nextNum = now + 2;`,
		},
		{
			name:      "non-spawn in a loop",
			reactorId: "9999904",
			body: "for (var i = 0; i < 3; i++) {\n" +
				"    rm.dropItems();\n" +
				"}",
			wantLine: `rm.dropItems();`,
		},
		{
			name:      "non-literal loop bound",
			reactorId: "9999905",
			body: "for (var i = 0; i < n; i++) {\n" +
				"    rm.spawnMonster(1);\n" +
				"}",
			wantLine: `for (var i = 0; i < n; i++) {`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := sourceScript{Id: tt.reactorId, ActBody: tt.body}
			_, err := convertScript(s)
			if err == nil {
				t.Fatalf("convertScript: expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.reactorId) {
				t.Errorf("error %q does not contain reactor id %q", err.Error(), tt.reactorId)
			}
			if !strings.Contains(err.Error(), tt.wantLine) {
				t.Errorf("error %q does not contain offending line %q", err.Error(), tt.wantLine)
			}
		})
	}
}
