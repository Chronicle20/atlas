# v48 Stage E — CLOSE batch E (character remnants + small producibles)

Anchor v61, IDB port 13337 (GMS_v48_1_DEVM.exe). Branch
`task-113-gms-legacy-versions`. Final pass over the v48 ❌ cells where the v61
anchor is verified.

## Result summary

**4 op cells promoted ❌ → ✅** (VIEW_ALL_CHAR cb, ADD_NEW_CHAR_ENTRY cb,
INVENTORY_OPERATION cb, WHISPER cb). **1 cell confirmed already-covered**
(NpcContinueConversation sb). **2 cells STOPPED cleanly with the full IDA
derivation captured** (SHOW_STATUS_INFO cb, GUILD_OPERATION cb) — both are
genuine multi-arm per-version dispatcher builds, not quick mirrors.

`matrix --check` exit 0; problem-grep 0; v48 conflicts 0. All 8 anchor versions
held (v83 367 / v84 345 / v87 379 / v95 399 / jms 362 / v72 216 / v79 228 /
v61 208). Every fixture byte traces to a live v48 decompile line.

v48 verified count 159 → 159 (flat): the 4 op-cell promotions are offset in the
raw count by the VIEW_ALL_CHAR fname-alignment collapsing 4 orphan
CharacterViewAll* sub-struct rows into the now-verified op (matching how v61
accounts them — a correctness improvement, not a regression).

## Per-cell outcomes

| Cell | Dir | Outcome |
|---|---|---|
| VIEW_ALL_CHAR | cb | ✅ registry fname `sub_50232D` → `CLogin::OnViewAllCharResult` (sub in fname_alts, mirrors v61 cb + v48 sb sibling); links op to the four CharacterViewAll* reports that closeB already fixtured |
| ADD_NEW_CHAR_ENTRY | cb | ✅ added the AddCharacterError v48 arm (worst-of-siblings); sub_501973 @0x501973 error branch reads only Decode1(code) then sub_50FF3B(18), no stat/avatar body — op14 + single [code] byte == v61 |
| INVENTORY_OPERATION | cb | ✅ CWvsContext::OnInventoryOperation @0x71a4f6 body-verified byte-identical to v79 @0x96953e; 5 arms (Add/ChangeMove/QuantityUpdate/Remove/ChangeBatch) == v79 (no gate below v79) |
| WHISPER | cb | ✅ CField::OnWhisper @0x4c71d5 body-verified == v61 @0x4eabd7; `Decode1(mode)-9` over modes 9/10/18/34; 7 arms mirror v61; Weather (146) v48-absent |
| NpcContinueConversation | sb | already covered — NPC_TALK_MORE op-cell is verified for v48; the standalone sub-struct ❌ row is a matrix gap-fill artifact (writer consumed by the verified op) |
| SHOW_STATUS_INFO | cb | **REMAINING** — full v48 OnMessage switch table derived (below); diverges from v61 modes, needs independent fixturing + dispositions |
| GUILD_OPERATION | cb | **REMAINING** — 41-arm CWvsContext::OnGuildResult family, zero v48 markers; dispatcher-family-implementer scale |

## Codec / registry changes (all leave v61/v72/v79/v83/v84/v87/v95/jms UNCHANGED)

1. `docs/packets/registry/gms_v48.yaml` — VIEW_ALL_CHAR cb: fname
   `sub_50232D` → `CLogin::OnViewAllCharResult`, `sub_50232D` moved to fname_alts.
2. `libs/atlas-packet/character/clientbound/add_entry_test.go` — added
   `TestAddCharacterErrorByteOutputV48` + marker + evidence.
3. `libs/atlas-packet/inventory/clientbound/change_v48_test.go` (new) — 5 arms.
4. `libs/atlas-packet/field/clientbound/whisper_test.go` — appended
   `TestWhisperByteOutputV48` + 7 markers.

No codec `.go` gates were needed for the promoted cells (all fast-path == v79/v61
for the legacy range).

## Commits (3, all staged explicitly; branch verified after each)

1. `314e936f3b` — VIEW_ALL_CHAR + ADD_NEW_CHAR_ENTRY cb.
2. `62c7de2709` — INVENTORY_OPERATION cb.
3. `0fbe7499f4` — WHISPER cb.

## EXACT remaining

### 1. SHOW_STATUS_INFO cb (v48 op 33 / 0x21) — CWvsContext::OnMessage @0x71b1b8

The v48 dispatcher has EXACTLY 10 arms (cases 0-9; default is a no-op). Full
switch table derived from IDA (port 13337):

| v48 case | sub | semantic | body after mode byte |
|---|---|---|---|
| 0 | sub_71B265 | DROP_PICKUP | Decode1(sub){-3,-2,-1,0,1,2}; sub 0→Decode4(id)+Decode4(cnt), sub 1→Decode4(meso)+Decode2, sub 2→Decode4(id), sub -1/-2/-3→none |
| 1 | sub_71B543 | QUEST_RECORD | Decode2(questId)+Decode1(sub){0,1,2}; 0→none, 1→DecodeStr, 2→DecodeBuffer(8) |
| 2 | sub_71B7D9 | CASH_ITEM_EXPIRE | Decode4(itemId) [GetItemName] |
| 3 | sub_71B9C0 | INCREASE_EXP | Decode1(white)+Decode4(exp)+Decode1(inChat)+Decode1(mobRate)+Decode1(partyRate)+[if mobRate>0: Decode1] |
| 4 | sub_71BD0C | INCREASE_FAME | Decode4(amount, signed) [str 277/278] |
| 5 | sub_71BDD8 | INCREASE_MESO | Decode4(amount, signed) [str 279/281] |
| 6 | sub_71BEA4 | INCREASE_GUILD_POINT | Decode4(amount, signed) [str 3192/3193] |
| 7 | sub_71BF70 | GENERAL_ITEM_EXPIRE | Decode4(itemId) [GetItemDesc] |
| 8 | sub_71B887 | (item-name list) | Decode1(count)+count×Decode4(itemId) [str 2580] — no clear Atlas struct |
| 9 | sub_71B96B | SYSTEM_MESSAGE | DecodeStr(message) |

**Why it is NOT a v61 mirror (off-by-one hazard — see MEMORY v83 bug):** v61's
OnMessage reserves case 4 for INCREASE_SKILL_POINT, shifting FAME=5/MESO=6/
GUILD_POINT=7/GIVE_BUFF=8. v48 has NO skill-point slot, so FAME=4/MESO=5/
GUILD_POINT=6, and v48 case 7 = GENERAL_ITEM_EXPIRE, case 8 = an item-name list
(v61 case 8 = GIVE_BUFF). So the v48 struct→mode assignment must be derived
independently (as above), NOT copied from v61.

Work to finish (next batch): fixture the 18 present structs with their v48 modes
(bodies mirror v61 for the <79 range — verify IncreaseExperience's <79 form byte-
for-byte against sub_71B9C0), disposition **StatusMessageGiveBuff** and
**StatusMessageIncreaseSkillPoint** n-a for v48 (no case reads a buff / SP), and
resolve case 8 (item-name list) — either map it to an existing struct or leave it
un-templated (no v48 report exists for it, so it is not a worst-of-siblings drag).
19 StatusMessage reports currently exist for v48; GiveBuff is the one that must be
stripped from siblings via `_unimplemented.json` + report removal.

### 2. GUILD_OPERATION cb (v48 op 53 / 0x35) — CWvsContext::OnGuildResult

41-arm dispatcher (v61 anchor arm count), zero v48 clientbound guild markers.
This is a full dispatcher-family build (dispatcher-family-implementer scale), not
a small producible. Worst-of-siblings requires all ~41 arms body-verified +
fixtured for v48. Deferred whole.

## Verification

- `go test` green: character/clientbound, inventory/clientbound, field/clientbound.
- `matrix --check` exit 0; problem-grep 0; v48 conflicts 0.
- Anchor counts held: v83 367 / v84 345 / v87 379 / v95 399 / jms 362 / v72 216 /
  v79 228 / v61 208. v48 159.
- Each commit staged explicitly (never `git add -A`); `git show --stat` scope
  limited to the target cells; no out-of-scope report-regen drift
  (AuthSuccess/ChatMulti/ReactorHitRequest/SUMMARY/MonsterCarnival untouched).
- Branch `task-113-gms-legacy-versions` verified after each commit.
