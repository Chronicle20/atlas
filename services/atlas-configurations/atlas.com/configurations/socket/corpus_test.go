package socket

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// The seed corpus is the strictest available proof that these blocking rules do
// not strand existing data: every rule here is a 400, so any seed template that
// fails Validate is a template the UI could never save back. Task-194 decision 1
// (strict validation) is only safe while this test passes.
func TestValidate_AcceptsEverySeedTemplate(t *testing.T) {
	dir := filepath.Join("..", "..", "..", "seed-data", "templates")
	files, err := filepath.Glob(filepath.Join(dir, "template_*.json"))
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	if len(files) != 11 {
		t.Fatalf("expected 11 seed templates, found %d in %s", len(files), dir)
	}

	type entry struct {
		OpCode    string   `json:"opCode"`
		Validator string   `json:"validator"`
		Handler   string   `json:"handler"`
		Writer    string   `json:"writer"`
		Services  []string `json:"services"`
	}
	type doc struct {
		Socket struct {
			Handlers []entry `json:"handlers"`
			Writers  []entry `json:"writers"`
		} `json:"socket"`
	}

	total := 0
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		var d doc
		if err := json.Unmarshal(b, &d); err != nil {
			t.Fatalf("parse %s: %v", f, err)
		}
		in := Input{}
		for _, h := range d.Socket.Handlers {
			in.Handlers = append(in.Handlers, Binding{Name: h.Handler, OpCode: h.OpCode, Validator: h.Validator, Services: h.Services})
		}
		for _, w := range d.Socket.Writers {
			in.Writers = append(in.Writers, Binding{Name: w.Writer, OpCode: w.OpCode, Services: w.Services})
		}
		total += len(in.Handlers) + len(in.Writers)
		if issues := Validate(in); len(issues) != 0 {
			t.Errorf("%s: %d issue(s):", filepath.Base(f), len(issues))
			for _, iss := range issues {
				t.Errorf("    %s: %s", iss.Path, iss.Message)
			}
		}
	}
	if total != 3217 {
		t.Errorf("corpus size = %d entries, want 3217 (3052 before task-206, plus task-206's 10 CashShopCouponCodeHandle bindings — every template but gms_12 — plus task-207's 7 CashItemGachaponHandle handlers and 6 CashItemGachaponResult writers — plus task-210's 16 template bindings (CharacterUseDeathItemHandle handler and CharacterShowUpgradeTombEffect writer in 8 templates) and 2 v92 writers (CharacterEffect and CharacterEffectForeign) — plus task-212's 15 catch bindings — plus task-211's 30 kite writer bindings (SpawnKite, SpawnKiteError and DestroyKite on every template but gms_12) — plus task-213's 1 gms_92 CharacterSkillPrepareHandle binding, the only template that lacked it — plus task-217's 12 Aran combo bindings (AranComboCounterHandle handler and ShowCombo writer on gms_83/84/87/92/95 and jms_185) — plus task-218's 3 CharacterKeyMapChangeHandle bindings on gms_87/gms_92/jms_185, the three templates that lacked it and where keybinds therefore never saved — plus task-221's 7 npc-shop bindings (NPCShopHandle handler on gms_87/92/95, plus the NPCShop and NPCShopOperation writers on gms_48 and gms_92, the two templates that lacked them) — plus task-226's 6 skill-macro bindings (CharacterSkillMacroHandle handler on gms_61, gms_87, gms_92, gms_95 and jms_185 — gms_61 included because task-226 corrected SKILL_MACRO x gms_v61 off n-a, having located CMacroSysMan::FlushToSvr at 0x59746c sending opcode 101 — plus the CharacterSkillMacro writer on gms_92) — plus task-224's 10 PetNameChanged writer bindings, one on every template but gms_12, carrying the CPet::OnNameChanged broadcast for the pet name tag) — plus task-230's 17 scripted-item bindings (ScriptedItemHandle on the 8 templates whose client carries SCRIPTED_ITEM — every template but gms_12, gms_48 and gms_61, the three where CWvsContext::SendScriptRunItemRequest does not exist — plus NpcItemUseHandle on the 9 templates carrying NPC_ITEM_USE_REQUEST, every template but gms_12 and gms_48, gms_61 included because task-230 located CWvsContext::SendSelectNpcItemUseRequest at 0x83778d there sending opcode 0x066) — plus task-228's 5 WaterOfLifeHandle handler bindings on gms_83/84/87/92/95, the five templates whose client sends the WATER_OF_LIFE opcode; the other six are n-a — plus the same task's 6 PetDestroyItemHandle bindings on gms_83/84/87/92/95 and jms_185, the six templates whose client sends DESTROY_PET_ITEM_REQUEST for a dried-up noRevive pet — plus task-229's 12 item-use bindings: CharacterItemUseSummonBagHandle on gms_87/92/95 and jms_185 and CharacterItemUseTownScrollHandle on gms_87/92/95 and jms_185, the four templates that lacked them, plus gms_48's new CharacterItemUseHandle at 0x38 (the pre-existing entry at 0x41 was rebound from CharacterItemUseHandle to CharacterItemUseTownScrollHandle against CWvsContext::SendPortalScrollUseRequest, so gms_48 is a net +1), plus gms_92's CharacterItemUseHandle, CharacterItemUseScrollHandle and PetFoodHandle, the ordinary item-use hole that made potions and scrolls inert on that column)", total)
	}
}
