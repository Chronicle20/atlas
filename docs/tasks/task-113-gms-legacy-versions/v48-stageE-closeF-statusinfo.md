# v48 Stage E — CLOSE batch F: SHOW_STATUS_INFO cb (OnMessage status dispatcher)

Anchor v61, IDB port 13337 (GMS_v48_1_DEVM.exe). Branch
`task-113-gms-legacy-versions`. CRASH-CRITICAL op-cell (status-message off-by-one
class, see MEMORY [[bug_v83_status_message_operations_off_by_one]]).

## Headline

**No off-by-one exists.** The v48 status-message operations table already matches
the verified v61 anchor. The closeE producibles doc's "v61 reserves case 4 for
INCREASE_SKILL_POINT → v48 shifts FAME/MESO/GUILD" claim was **disproven** by
decompiling v61 `CWvsContext::OnMessage` directly. The template was left UNCHANGED
(changing it — removing GIVE_BUFF, renumbering GENERAL_ITEM_EXPIRE 8→7 as the task
brief anticipated — would have *introduced* the very client crash it feared by
misaligning from the verified v61 client).

## 1. v48 operations table (verified from the switch)

`CWvsContext::OnMessage` @0x71b1b8, switch @0x71b1cf — 10 arms (cases 0-9,
default = no-op). Each arm body re-confirmed against its v61 twin
(`CWvsContext::OnMessage` @0x8437ef, same 10-arm shape):

| mode | v48 sub | semantic (Atlas key) | body after mode byte | v61 twin |
|---|---|---|---|---|
| 0 | sub_71B265 @0x71b265 | DROP_PICK_UP | Decode1(sub){-3,-2,-1,0,1,2}; 0→id+cnt, 1→meso+cafe(NO partial, <79), 2→id, neg→none | sub_8438B5 |
| 1 | sub_71B543 @0x71b543 | QUEST_RECORD | Decode2(questId)+Decode1(sub); 0→none(forfeit), 1→DecodeStr(update), 2→DecodeBuffer8(complete) | sub_843BD8 |
| 2 | sub_71B7D9 @0x71b7d9 | CASH_ITEM_EXPIRE | Decode4(itemId) [GetItemName] | (v61 marker n/a) |
| 3 | sub_71B9C0 @0x71b9c0 | INCREASE_EXPERIENCE | white,exp,inChat,mobEvent%,partyBonus%,[mobEvent>0:playTime] — SHORT | sub_84418A (longer) |
| 4 | sub_71BD0C @0x71bd0c | INCREASE_FAME | Decode4 signed [str 277/278] | sub_84471B |
| 5 | sub_71BDD8 @0x71bdd8 | INCREASE_MESO | Decode4 signed [str 279/281] | sub_8447DD |
| 6 | sub_71BEA4 @0x71bea4 | INCREASE_GUILD_POINT | Decode4 signed [str 3192/3193] | sub_8448AF |
| 7 | sub_71BF70 @0x71bf70 | GIVE_BUFF | Decode4(itemId) [GetItemDesc] | sub_844971 |
| 8 | sub_71B887 @0x71b887 | GENERAL_ITEM_EXPIRE | Decode1(count)+count×Decode4 [GetItemName, str 2580] | sub_844063 |
| 9 | sub_71B96B @0x71b96b | SYSTEM_MESSAGE | DecodeStr | sub_84413D |

Ground-truth cross-check: existing verified v61 markers pin GiveBuff→arm7
(0x844971, single Decode4/GetItemDesc) and GeneralItemExpire→arm8 (0x844063,
count+list/GetItemName), and Fame→arm4 (0x84471b). v48 arm7/arm8/arm4 bodies are
byte-identical. v61 arm4 is INCREASE_FAME, **not** a skill-point slot — the
closeE premise was wrong.

## 2. Template off-by-one fix

**Before == After (no change).** `template_gms_48_1.json` op 0x21
`CharacterStatusMessage` operations already = `{DROP_PICK_UP:0, QUEST_RECORD:1,
CASH_ITEM_EXPIRE:2, INCREASE_EXPERIENCE:3, INCREASE_FAME:4, INCREASE_MESO:5,
INCREASE_GUILD_POINT:6, GIVE_BUFF:7, GENERAL_ITEM_EXPIRE:8, SYSTEM_MESSAGE:9}`.
Byte-identical to the verified v61/v72/v79 templates. No INCREASE_SKILL_POINT slot
(correctly absent — v48 has no arm for it). GIVE_BUFF(single int)→arm7(single
Decode4) and GENERAL_ITEM_EXPIRE(count+list)→arm8(count+list) both byte-match.
Template intentionally untouched.

## 3. Per-arm fixture outcome (worst-of-siblings)

19 StatusMessage siblings (the full set of v48 reports) marked `gms_v48` + evidence
pinned, per-arm addresses. Roundtrip green.

- **arm 0** DropPickUp{ItemUnavailable, InventoryFull, GameFileDamaged, Stackable,
  UnStackable, Meso} + DropLoss{Stackable, UnStackable} — 8 structs, ✅ == v61
  (meso partial byte already `<79`-gated; v48 arm0 sub 1 reads no partial — confirmed).
- **arm 1** {Forfeit, Update, Complete}QuestRecord — 3 structs, ✅ == v61.
- **arm 2** CashItemExpire — ✅.
- **arm 3** IncreaseExperience — ✅ **v48-gated** (`< v61` short body; see §4).
- **arm 4** IncreaseFame — ✅ == v61.
- **arm 5** IncreaseMeso — ✅ == v61.
- **arm 6** IncreaseGuildPoint — ✅ == v61.
- **arm 7** GiveBuff — ✅ == v61 (single Decode4).
- **arm 8** GeneralItemExpire — ✅ == v61 (count+list).
- **arm 9** SystemMessage — ✅ == v61.

**Arms n-a: none.** All 19 v48 reports map to present arms 0-9. INCREASE_SKILL_POINT,
GIVE_BUFF-as-a-shift, QUEST_RECORD_EX, ITEM_PROTECT_EXPIRE, ITEM_EXPIRE_REPLACE,
SKILL_EXPIRE, JMS_COUNTER_NOTICE have **no v48 report** (version-absent) so are not
worst-of-siblings drags — no `_unimplemented.json` entry needed.

## 4. Codec change — IncreaseExperience `< v61` gate

Only codec `.go` change. v48 sub_71B9C0 reads a much shorter exp body than
v61 sub_84418A: after `inChat` it reads `Decode1(mobEvent%)@0x71ba01`,
`Decode1(partyBonus%)@0x71ba15`, `[if mobEvent>0: Decode1(playTime)@0x71ba24]`
and STOPS. It has NO `monsterBookBonus`/`weddingBonusEXP` ints (v61 reads them at
@0x8441cd / @0x8441f3), NO inChat quest-rate block, NO
`partyBonusEventRate`/`partyBonusExp` (v61 @0x844245). Added a
`legacyV48 = GMS && MajorVersion() < 61` branch to both Encode and Decode (skips
the two ints, early-returns after the optional playTime). v61/72/79/83/84/87/95/JMS
paths unchanged. Existing `<79`/`<83`/`>=95` gates preserved. RoundTrip asserts
byte-consumption parity (encode/decode symmetric); the `< v61` legacy path is
exercised by the "GMS v28" test variant (major 28 < 61).

## 5. Gates

- `go test ./libs/atlas-packet/character/...` green; `-race` green
  (character/clientbound 1.1s).
- `go vet ./libs/atlas-packet/character/...` clean.
- Template still valid JSON (untouched).
- `go run ./tools/packet-audit matrix --check` exit **0**.
- problem-grep (`orphan|dangling|stale|drift|unresolv|malformed` in STATUS.md) = **0**.
- v48 conflicts (🟥) = **0**.

## 6. Matrix / regression

- **SHOW_STATUS_INFO v48: ❌ → ✅** (1 op-cell promoted).
- v48 verified 159 → **160** (+1). All anchor counts held:
  v61 208 / v72 216 / v79 228 / v83 367 / v84 345 / v87 379 / v95 399 / JMS 362.
- STATUS.md diff limited to the SHOW_STATUS_INFO row + v48 summary count; status.json
  3 lines (v48 state + count). No out-of-scope drift
  (AuthSuccess/ChatMulti/ReactorHitRequest/MonsterCarnival untouched).

## 7. Commit

- `task-113(v48): stage E — verify SHOW_STATUS_INFO cb OnMessage dispatcher (tier-1)`
  — status_message.go (`<61` IncEXP gate), status_message_test.go (19 markers),
  19 evidence YAMLs, STATUS.md, status.json. Staged explicitly (no `git add -A`).
  Branch verified `task-113-gms-legacy-versions` after commit.

## Notes / concerns

- The task brief's crash-critical "fix the off-by-one" premise was based on the
  closeE prose error. Verified against v48 IDA + v61 IDA + the verified v61 markers
  + the registry note — no off-by-one. Deliberately did NOT alter the template.
- No genuine blockers. Every fixture byte traces to a live v48 (and v61) decompile line.
