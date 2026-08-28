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
	if total != 3387 {
		t.Errorf("corpus size = %d entries, want 3387 (3052 before task-206, plus task-206's 10 CashShopCouponCodeHandle bindings — every template but gms_12 — plus task-207's 7 CashItemGachaponHandle handlers and 6 CashItemGachaponResult writers — plus task-210's 16 template bindings (CharacterUseDeathItemHandle handler and CharacterShowUpgradeTombEffect writer in 8 templates) and 2 v92 writers (CharacterEffect and CharacterEffectForeign) — plus task-212's 15 catch bindings — plus task-211's 30 kite writer bindings (SpawnKite, SpawnKiteError and DestroyKite on every template but gms_12) — plus task-213's 1 gms_92 CharacterSkillPrepareHandle binding, the only template that lacked it — plus task-217's 12 Aran combo bindings (AranComboCounterHandle handler and ShowCombo writer on gms_83/84/87/92/95 and jms_185) — plus task-218's 3 CharacterKeyMapChangeHandle bindings on gms_87/gms_92/jms_185, the three templates that lacked it and where keybinds therefore never saved — plus task-221's 7 npc-shop bindings (NPCShopHandle handler on gms_87/92/95, plus the NPCShop and NPCShopOperation writers on gms_48 and gms_92, the two templates that lacked them) — plus task-226's 6 skill-macro bindings (CharacterSkillMacroHandle handler on gms_61, gms_87, gms_92, gms_95 and jms_185 — gms_61 included because task-226 corrected SKILL_MACRO x gms_v61 off n-a, having located CMacroSysMan::FlushToSvr at 0x59746c sending opcode 101 — plus the CharacterSkillMacro writer on gms_92) — plus task-224's 10 PetNameChanged writer bindings, one on every template but gms_12, carrying the CPet::OnNameChanged broadcast for the pet name tag) — plus task-230's 17 scripted-item bindings (ScriptedItemHandle on the 8 templates whose client carries SCRIPTED_ITEM — every template but gms_12, gms_48 and gms_61, the three where CWvsContext::SendScriptRunItemRequest does not exist — plus NpcItemUseHandle on the 9 templates carrying NPC_ITEM_USE_REQUEST, every template but gms_12 and gms_48, gms_61 included because task-230 located CWvsContext::SendSelectNpcItemUseRequest at 0x83778d there sending opcode 0x066) — plus task-228's 5 WaterOfLifeHandle handler bindings on gms_83/84/87/92/95, the five templates whose client sends the WATER_OF_LIFE opcode; the other six are n-a — plus the same task's 6 PetDestroyItemHandle bindings on gms_83/84/87/92/95 and jms_185, the six templates whose client sends DESTROY_PET_ITEM_REQUEST for a dried-up noRevive pet — plus task-227's 67 cash-shop name-change/world-transfer bindings: CashShopCheckNameChangePossibleHandle handler and CashShopCheckNameChange writer on every template but gms_12 and jms_185 (9 each); CashShopCheckTransferWorldPossibleHandle handler and CashShopCheckTransferWorldPossibleResult writer on every template but gms_12 (10 each); CashShopCancelNameChangeResult and CashShopCancelTransferWorldResult writers on every template but gms_12/gms_48 (8 each); CancelNameChangeByOther writer on every template but gms_12/gms_48/gms_61 (7); CashShopCheckNameChangePossibleResult writer on gms_79/83/84/87/92/95 (6) — plus this task's 9 CashShopCheckNameChangeHandle bindings, one on each GMS template (gms_48 at 0x11, the other eight at 0x15): the channel-scoped half of CHECK_CHAR_NAME, whose opcode the client uses for BOTH CLogin::SendCheckDuplicateIDPacket and CCashShop::SendCheckDuplicateIDPacket, so the two bindings coexist at one opcode with disjoint services. jms_185 is excluded — it has no name-change feature at all — plus task-229's 12 item-use bindings: CharacterItemUseSummonBagHandle and CharacterItemUseTownScrollHandle on gms_87/92/95 and jms_185, the four templates that lacked them (8); plus gms_48's new CharacterItemUseHandle at 0x38, where the pre-existing 0x41 entry was rebound from CharacterItemUseHandle to CharacterItemUseTownScrollHandle against CWvsContext::SendPortalScrollUseRequest, so gms_48 is a net +1; plus gms_92's CharacterItemUseHandle, CharacterItemUseScrollHandle and PetFoodHandle (3), the ordinary item-use hole that made potions and scrolls inert on that column — plus task-225's 24 dragon bindings (the DragonMoveHandle handler plus the DragonSpawn, DragonMove and DragonRemove writers on gms_83/84/87/92/95 and jms_185, the six templates whose client has a CDragon) — plus task-241's 16 Duey parcel-delivery bindings: the DueyActionHandle handler (gated by LoggedInValidator, the narrowest validator that exists in this corpus — DUEY_ACTION is sent from an in-game NPC dialogue, so it is post-login, not pre-login like NoOpValidator's users) and the Parcel writer, one pair on each of gms_72/79/83/84/87/92/95 and jms_185, the eight templates whose client carries the Duey NPC parcel-delivery feature — plus task-255's 10 AutoAggro handler bindings, one on every template but gms_12, carrying CMob::ApplyControl, the client's auto-aggro request, into the channel service) — plus task-249's 8 TouchReactorHandle bindings on gms_72/79/83/84/87/92/95 and jms_185, the eight templates whose client sends the TOUCHING_REACTOR opcode; gms_12, gms_48 and gms_61 are n-a — plus task-250's 10 InnerPortalHandle handler bindings, one on every template but gms_12 (which has no gms_v12.yaml registry at all, so it is structurally unrouted rather than an oversight), carrying the intra-map portal entry request, USE_INNER_PORTAL — plus task-246's 12 Maple Life bindings (MapleLifeCheckNameHandle handler and the MapleLifeResult and MapleLifeError writers on gms_83/87/92/95, the four templates whose client carries CUICharacterSaleDlg) — plus task-240's 1 CashShopOpen writer binding on gms_95, the only template that lacked it, carrying the CStage::OnSetCashShop cash-shop open packet) — plus task-146's 10 further gms_v95.1 bindings: 7 new channel handlers routed by the v95 packet-verification batch (MapChangeHandle at 0x29, CashShopEntryHandle at 0x2B, NPCStartConversationHandle at 0x3F, CharacterAutoDistributeApHandle at 0x63, PortalScriptHandle at 0x70, CharacterMultiChatHandle at 0x8C and PartyInviteRejectHandle at 0x92) and 2 new writers (PicResult on the login service at 0x1B and NPCConversation at 0x16B) in the same pass; task-146's third v95 writer, CashShopOpen at 0x8F, is already counted above under task-240 — that pass also rebound 0x19 from NoOpHandler to PongHandle for CClientSocket::OnAliveReq, reusing the existing slot and adding no corpus entry — plus a further StorageOperationHandle handler at 0x43, added when the party/guild/buddy and messenger/BBS/storage v95 serverbound mode tables were routed; all 10 land on gms_v95.1, the only template this batch touches) — plus the task-246 bug fix's 3 gms_84 Maple Life bindings (MapleLifeCheckNameHandle handler and the MapleLifeResult and MapleLifeError writers), correcting derivation.md §2.0's wrong VERSION-ABSENT finding: CUICharacterSaleDlg does exist on gms_v84, at its own opcodes (263/359/360) read off the v84 CUICharacterSaleDlg::OnPacket dispatcher, so gms_84 now carries the same three bindings as its gms_83/87/92/95 neighbours)", total)
	}
}
