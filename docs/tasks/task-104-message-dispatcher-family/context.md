# task-104 — Context & Key Facts

Companion to `plan.md`. Read this first; it captures the grounded facts the plan depends on
so an executor with zero prior context can work safely.

## What this task is (and is NOT)

`CWvsContext::OnMessage` (the MESSAGE / `CHARACTER_STATUS_MESSAGE` opcode, `CWvsContext::OnPacket`
case 0x26) is a **mode-prefix dispatcher**: the client reads a leading mode byte and routes to a
per-mode sub-handler. Two arms fan out further on an inner discriminator
(`OnDropPickUpMessage`, `OnQuestRecordMessage`). Atlas already implements all 24 arms.

**Unlike guild (task-103), the codec + config-driven body layer is ALREADY built and footgun-free:**

- 24 discrete `StatusMessage*` structs in
  `libs/atlas-packet/character/clientbound/status_message.go`, each constructor takes `mode byte`
  (zero `mode: 0x` literals).
- 23 body funcs in `libs/atlas-packet/character/status_message_body.go`, each
  `WithResolvedCode("operations", "<FIXED_KEY>", func(mode byte) …)` — no caller selector
  (INV-2/INV-3 clean). 23 funcs → 24 structs because `…DropLossItemBody` constructs the
  stackable-or-unstackable struct by inventory type.
- ~8 atlas-channel consumers already call these body funcs.

So this is **NOT** a codec refactor. It is three things:
1. **Fix the v83 mode-table correctness bug** (the triggering bug — see below).
2. **Enroll the family in the dispatcher-audit tooling** (24 `#`-entries) so each arm is
   individually byte-verified and the bug can't recur.
3. **Resolve the jms ❌** via export decomposition.

## The triggering bug (IDA-grounded this session)

The committed `docs/packets/dispatchers/character_status_message.yaml` claims the 15-mode table
is "version-STABLE … byte-identical in v83 and v95." **That claim is false**, and the committed
gms_83 seed template (`template_gms_83_1.json:1450`) carries the wrong table.

Decompiled this session (addresses are the `CWvsContext::OnMessage` switch per version):

| version | port | switch addr | case count | SP arm? |
|---|---|---|---|---|
| gms_v83 | 13342 | `0xA209D4` | 14 (0–0xD) | **absent** |
| gms_v84 | 13337 | `0xA6BDD9` | 15 (0–0xE) | present (case 4) — `sub_*`, confirm per-arm |
| gms_v87 | 13341 | `0xAB8076` | 15 (0–0xE) | present (case 4) — named |
| gms_v95 | 13340 | `0xA06C90` | 15 (0–0xE) | present (case 4) — named (authoritative) |
| jms_v185 | 13339 | `0xB078F3` | **16 (0–0xF)** | present; case 0xF = `sub_B0931C` (no Atlas arm) |

v95 PDB names are authoritative: case 4 `OnIncSPMessage`, 5 `OnIncPOPMessage` (fame), 6
`OnIncMoneyMessage` (meso), 7 `OnIncGPMessage`, 8 `OnGiveBuffMessage`, … 14 `OnSkillExpireMessage`.

**Consequence on a live v83 tenant today:** Atlas sends FAME at mode 5, but v83 mode 5 is
`OnIncMoneyMessage` → fame renders as meso; every arm from fame up is off by one; SKILL_EXPIRE
(sent at 14) exceeds v83's `default` boundary (0xD) and is dropped; INCREASE_SKILL_POINT (sent at
4) hits v83's fame handler. This is the same class as
`bug_v83_status_message_operations_off_by_one` in project memory.

## The corrected per-version mode table (SOURCE OF TRUTH)

v83 ≠ v84+ because v84 inserts `OnIncSPMessage` at case 4, shifting everything from fame up by one.

| key (outer mode) | gms_v83 | gms_v84 | gms_v87 | gms_v95 | jms_v185 |
|---|---|---|---|---|---|
| DROP_PICK_UP | 0 | 0 | 0 | 0 | 0 |
| QUEST_RECORD | 1 | 1 | 1 | 1 | 1 |
| CASH_ITEM_EXPIRE | 2 | 2 | 2 | 2 | 2 |
| INCREASE_EXPERIENCE | 3 | 3 | 3 | 3 | 3 |
| INCREASE_SKILL_POINT | *(omit)* | 4 | 4 | 4 | 4 |
| INCREASE_FAME | 4 | 5 | 5 | 5 | 5 |
| INCREASE_MESO | 5 | 6 | 6 | 6 | 6 |
| INCREASE_GUILD_POINT | 6 | 7 | 7 | 7 | 7 |
| GIVE_BUFF | 7 | 8 | 8 | 8 | 8 |
| GENERAL_ITEM_EXPIRE | 8 | 9 | 9 | 9 | 9 |
| SYSTEM_MESSAGE | 9 | 10 | 10 | 10 | 10 |
| QUEST_RECORD_EX | 10 | 11 | 11 | 11 | 11 |
| ITEM_PROTECT_EXPIRE | 11 | 12 | 12 | 12 | 12 |
| ITEM_EXPIRE_REPLACE | 12 | 13 | 13 | 13 | 13 |
| SKILL_EXPIRE | 13 | 14 | 14 | 14 | 14 |

"omit" = the `INCREASE_SKILL_POINT` key has **no `gms_v83` entry in its `modes` map** (absence,
not a fabricated byte). v84 SP-at-4 is the strong prior but is verified per-arm, not assumed (D8).

## The 24 arms (struct → operations key → inner discriminator)

DROP_PICK_UP family (outer mode 0 every version; inner int8 written by `Encode`):

| # | struct | inner disc. | body func |
|---|---|---|---|
| 1 | StatusMessageDropPickUpItemUnavailable | `-2` | `…DropPickUpItemUnavailableBody()` |
| 2 | StatusMessageDropPickUpInventoryFull | `-1` | `…DropPickUpInventoryFullBody()` |
| 3 | StatusMessageDropPickUpGameFileDamaged | `-3` | `…DropPickUpGameFileDamagedBody()` |
| 4 | StatusMessageDropPickUpStackableItem | `0` | `…OperationDropPickUpStackableItemBody(itemId,amount)` |
| 5 | StatusMessageDropPickUpUnStackableItem | `2` | `…OperationDropPickUpUnStackableItemBody(itemId)` |
| 6 | StatusMessageDropLossStackableItem | `0` (neg qty) | `…OperationDropLossItemBody(itemId,qty)` (by inv type) |
| 7 | StatusMessageDropLossUnStackableItem | `2` | `…OperationDropLossItemBody(itemId,qty)` (by inv type) |
| 8 | StatusMessageDropPickUpMeso | `1` | `…OperationDropPickUpMesoBody(partial,amount,bonus)` |

QUEST_RECORD family (outer mode 1; inner byte after questId):

| # | struct | inner disc. | body func |
|---|---|---|---|
| 9 | StatusMessageForfeitQuestRecord | `0` | `…OperationForfeitQuestRecordBody(questId)` |
| 10 | StatusMessageUpdateQuestRecord | `1` | `…OperationUpdateQuestRecordBody(questId,info)` |
| 11 | StatusMessageCompleteQuestRecord | `2` | `…OperationCompleteQuestRecordBody(questId,completedAt)` |

Singleton arms (one outer mode each, no inner fan-out):

| # | struct | key | body func |
|---|---|---|---|
| 12 | StatusMessageCashItemExpire | CASH_ITEM_EXPIRE | `…OperationCashItemExpireBody(itemId)` |
| 13 | StatusMessageIncreaseExperience | INCREASE_EXPERIENCE | `…OperationIncreaseExperienceBody(…)` |
| 14 | StatusMessageIncreaseSkillPoint | INCREASE_SKILL_POINT | `…OperationIncreaseSkillPointBody(jobId,amount)` |
| 15 | StatusMessageIncreaseFame | INCREASE_FAME | `…OperationIncreaseFameBody(amount)` |
| 16 | StatusMessageIncreaseMeso | INCREASE_MESO | `…OperationIncreaseMesoBody(amount)` |
| 17 | StatusMessageIncreaseGuildPoint | INCREASE_GUILD_POINT | `…OperationIncreaseGuildPointBody(amount)` |
| 18 | StatusMessageGiveBuff | GIVE_BUFF | `…OperationGiveBuffBody(itemId)` |
| 19 | StatusMessageGeneralItemExpire | GENERAL_ITEM_EXPIRE | `…OperationGeneralItemExpireBody(itemIds)` |
| 20 | StatusMessageSystemMessage | SYSTEM_MESSAGE | `…OperationSystemMessageBody(message)` |
| 21 | StatusMessageQuestRecordEx | QUEST_RECORD_EX | `…OperationQuestRecordExBody(questId,info)` |
| 22 | StatusMessageItemProtectExpire | ITEM_PROTECT_EXPIRE | `…OperationItemProtectExpireBody(itemIds)` |
| 23 | StatusMessageItemExpireReplace | ITEM_EXPIRE_REPLACE | `…OperationItemExpireReplaceBody(messages)` |
| 24 | StatusMessageSkillExpire | SKILL_EXPIRE | `…OperationSkillExpireBody(skillIds)` |

## Critical correction: D5 (v83 SP) — `ResolveCode` does NOT no-op

`libs/atlas-packet/resolve.go:27` `ResolveCode` **returns 99 and still encodes** when a key is
absent (logs an error). It does NOT no-op. The design's "resolves to no-op" wording is wrong.

The real safeguard for v83 SP: **no consumer calls the SP body func.**
`grep -rn "IncreaseSkillPointBody\|NewStatusMessageIncreaseSkillPoint" services/ libs/` (excluding
the definition + tests) returns **zero** call sites. SP is emitted only by a future v84+ caller.
On v83 the key is simply absent → if anything ever did call it on v83 it would log-and-send 99,
which is acceptable because v83 genuinely has no SP arm. Plan task verifies this stays true.

## The exemplars to copy (in THIS worktree, dispatcher-lint-clean)

- **`CField::OnFieldEffect`** (`run.go` ~line 1823) — discrete `#`-entries, **no bare-root case**,
  structs in `field/clientbound/effect.go`. This is the model for D2.
- **`CITC::OnNormalItemResult`** (MTS_OPERATION, `run.go` ~line 1886+) — 35 `#`-arms, per-mode
  body funcs in `field/mts_operation_body.go`, **retired the phantom single-rep** explicitly.

⚠️ **Do NOT copy guild as the run.go root model.** In this worktree guild is migrated to
`#`-entries BUT is still in `dispatcher-lint-baseline.yaml` (its `GuildErrorBody(code)` catch-all
still trips INV-3). Its bare root returns a representative. FIELD_EFFECT/MTS are the clean models:
**no bare-root case at all.**

## operations table mechanics

- Source of truth = `docs/packets/dispatchers/character_status_message.yaml`
  (`Modes map[string]int`). Omitting a version key from a `modes` map = that version is absent.
- `go run ./tools/packet-audit operations` (no `--check`) **generates** the seed-template
  `operations` maps FROM the yaml across all 5 templates.
- `go run ./tools/packet-audit operations --check` exits 1 on drift / missing table / extra-stale
  keys. This is the gate.
- The 5 templates: `services/atlas-configurations/seed-data/templates/template_{gms_83_1,gms_84_1,gms_87_1,gms_95_1,jms_185_1}.json`.

## Verification shape (no golden bytes)

Verified fixtures use `test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)` (encode→decode,
assert zero leftover bytes) — NOT exact-byte golden comparison. All 24 round-trip tests already
exist in `status_message_test.go`. The new work is **per-arm `// packet-audit:verify` markers**
with real per-version IDA addresses (v83 omits the SP marker). `test.Variants`
(`libs/atlas-packet/test/context.go:18`) covers GMS v28/v83/v87/v95/v86/v84 + JMS v185; only the
5 registry versions get markers.

## Matrix / row aggregation

Registry op row is `SHOW_STATUS_INFO` (fname `CWvsContext::OnMessage`), `STATUS.md:61`, currently
✅ v83/v84/v87/v95 + ❌ jms. After migration, `baseFName` strips the `#` suffix so all 24 arm
reports collapse onto that op row, graded **worst-of** (the FIELD_EFFECT/MTS model). No `#Mode`
representative; **delete** the bare `case "CWvsContext::OnMessage":` block (`run.go:392`).

## Ports / tools

- IDA via ida-pro-mcp: `select_instance(port)` then `func_query`/`decompile`. Ports: v83=13342,
  v84=13337, v87=13341, v95=13340, jms=13339. Confirm version before reading (CLAUDE.md).
- jms delegate addresses captured this session: `OnDropPickUpMessage 0xB07A01` …
  `OnSkillExpireMessage 0xB088A4`, `sub_B0931C 0xB0931C`. Per-delegate addresses for the other
  jms arms and for all GMS arms must be decompiled and recorded during enumeration.

## Stop-and-ask gates (do NOT invent past these)

- **jms `sub_B0931C` (mode 0xF)** (D7): resolve its real name/arm from the jms IDB. If it maps to
  an existing Atlas arm, wire it; if it's a jms-only arm with no Atlas equivalent, **STOP and ask**
  — do not invent a struct, fake an fname, or grade it. jms ✅ for the 15 shared arms does not
  depend on 0xF.
- Any unresolved packet-audit fname → stop-and-ask (FR-1.3,
  `feedback_unresolved_fname_escalate`).

## Gates that must exit 0 at done

From the worktree root:
1. `go run ./tools/packet-audit dispatcher-lint`
2. `go run ./tools/packet-audit matrix --check`
3. `go run ./tools/packet-audit fname-doc --check`
4. `go run ./tools/packet-audit operations --check`
5. `go build ./... && go vet ./... && go test -race ./...` in `libs/atlas-packet`,
   `tools/packet-audit`, `services/atlas-channel`
6. `docker buildx bake atlas-channel` (no go.mod touched in atlas-channel expected, but the body
   layer lives in libs/atlas-packet which atlas-channel consumes — bake to be safe per CLAUDE.md)
7. `tools/redis-key-guard.sh` (no Redis expected; run for completeness)

## Out of scope

No business-logic change to consumers (call-site verify only). No new arms (jms 0xF escalated, not
invented). No DB/REST change. No new tenant version / LB ports. Migrating party/buddy/guild fully
off the baseline is a separate cycle.

## Enumeration results

Grounded from live IDA decompiles this session (Task 1). Every address below was read directly
from the named IDB; ports/binaries confirmed against `list_instances` before reading:
v83=13342 `MapleStory_dump.exe`, v84=13337 `GMS_v84.1_U_DEVM.exe`, v87=13341 `GMSv87_4GB.exe`,
v95=13340 `GMS_v95.0_U_DEVM.exe`, jms=13339 `MapleStory_dump_SCY.exe` (JMS v185).

### 1. Per-version `CWvsContext::OnMessage` switch confirmation

| version | switch addr | case range | case count | SP arm (case 4)? |
|---|---|---|---|---|
| gms_v83 | `0xA209D4` | 0–0xD | 14 | **absent** — case 4 is `OnIncPOPMessage` (fame) |
| gms_v84 | `0xA6BDD9` | 0–0xE | 15 | present (`sub_A6CEFA`, verified = SP) |
| gms_v87 | `0xAB8076` | 0–0xE | 15 | present (`OnIncSPMessage`) |
| gms_v95 | `0xA06C90` | 0–0xE | 15 | present (`OnIncSPMessage`, PDB-named, authoritative) |
| jms_v185 | `0xB078F3` | 0–0xF | **16** | present (`OnIncSPMessage`); **case 0xF = `sub_B0931C` (jms-only `StatusMessageJMSCounterNotice` / `JMS_COUNTER_NOTICE` — RESOLVED)** |

All switches read the outer mode via `CInPacket::Decode1` and `return` on `default`
(no out-of-range handler). This confirms the corrected mode table in this doc: v83 fame=4/meso=5/
GP=6/buff=7/...; v84+ insert SP at 4 shifting fame=5/meso=6/GP=7/buff=8/... No decompiled value
differs from the expected table.

### 2. Per-version, per-arm delegate addresses (canonical source for Tasks 4/5/8)

Outer mode = the switch case. v83 has no SP row (mode 4 is fame). Names are the IDB's own
(v84 are `sub_*` — semantically verified per-arm in §4).

| arm (operations key) | gms_v83 | gms_v84 | gms_v87 | gms_v95 | jms_v185 |
|---|---|---|---|---|---|
| DROP_PICK_UP (mode 0) `OnDropPickUpMessage` | `0xA20AD9` | `0xA6BEEF` | `0xAB818C` | `0x9FE190` | `0xB07A01` |
| QUEST_RECORD (mode 1) `OnQuestRecordMessage` | `0xA20F4C` | `0xA6C362` | `0xAB85D2` | `0xA03920` | `0xB07E49` |
| CASH_ITEM_EXPIRE `OnCashItemExpireMessage` | `0xA216FC` | `0xA6CB31` | `0xAB8D8E` | `0x9F8060` | `0xB085DF` |
| INCREASE_EXPERIENCE `OnIncEXPMessage` | `0xA21AC5` | `0xA6CFD7` | `0xAB9234` | `0x9F86C0` | `0xB08A97` |
| INCREASE_SKILL_POINT `OnIncSPMessage` | *(absent)* | `0xA6CEFA` | `0xAB9157` | `0x9F8570` | `0xB089AB` |
| INCREASE_FAME `OnIncPOPMessage` | `0xA2212D` | `0xA6D63F` | `0xAB9975` | `0x9F90A0` | `0xB09180` |
| INCREASE_MESO `OnIncMoneyMessage` | `0xA221F3` | `0xA6D705` | `0xAB9A3B` | `0x9FE910` | `0xB09246` |
| INCREASE_GUILD_POINT `OnIncGPMessage` | `0xA222C9` | `0xA6D7DB` | `0xAB9B11` | `0x9F91E0` | `0xB09397` |
| GIVE_BUFF `OnGiveBuffMessage` | `0xA2238F` | `0xA6D8A1` | `0xAB9BD7` | `0x9F2DF0` | `0xB0945D` |
| GENERAL_ITEM_EXPIRE `OnGeneralItemExpireMessage` | `0xA217A2` | `0xA6CBD7` | `0xAB8E34` | `0x9F8180` | `0xB08686` |
| SYSTEM_MESSAGE `OnSystemMessage` | `0xA21A78` | `0xA6CEAD` | `0xAB910A` | `0x9FE860` | `0xB0895E` |
| QUEST_RECORD_EX `OnQuestRecordExMessage` | `0xA2160B` | `0xA6CA40` | `0xAB8C9D` | `0x9FE6A0` | `0xB084EE` |
| ITEM_PROTECT_EXPIRE `OnItemProtectExpireMessage` | `0xA2187E` | `0xA6CCB3` | `0xAB8F10` | `0x9F82E0` | `0xB08763` |
| ITEM_EXPIRE_REPLACE `OnItemExpireReplaceMessage` | `0xA2195A` | `0xA6CD8F` | `0xAB8FEC` | `0x9FE7A0` | `0xB08840` |
| SKILL_EXPIRE `OnSkillExpireMessage` | `0xA219BE` | `0xA6CDF3` | `0xAB9050` | `0x9F8440` | `0xB088A4` |
| *(jms-only mode 0xF)* `sub_B0931C` | — | — | — | — | `0xB0931C` **RESOLVED: implemented as `StatusMessageJMSCounterNotice` / key `JMS_COUNTER_NOTICE` / jms mode 15 / delegate `0xB0931C` / wire `[mode][int32]` / semantics: single int → localized StringPool 5603, chat-type-6. message text runtime-encrypted, name is structural.** |

The 24 Atlas arms collapse onto the 15 operations keys above: DROP_PICK_UP fans out to 8 structs
(inner int8), QUEST_RECORD fans out to 3 (inner byte), the other 13 keys are singletons. The
delegate address is per *key/mode*; per-arm verification of the two fan-out families uses the
mode-0 and mode-1 delegate plus the inner discriminator in §3.

### 3. Inner fan-out discriminators (decompile-confirmed, stable across all 5 versions)

**OnDropPickUpMessage** (mode 0): reads `Decode1` → inner int8 `v3`, then:

| inner disc | branch in decompile | Atlas arm |
|---|---|---|
| `1` | `if (v3 == 1)` → `Decode1` partial, `Decode4` amount, `Decode2` bonus | DropPickUpMeso |
| `0` | `if (v3)` else / `case 0` → `Decode4` itemId, `Decode4` qty | DropPickUpStackableItem / DropLossStackableItem (neg qty) |
| `2` | `case 2` → `Decode4` itemId | DropPickUpUnStackableItem / DropLossUnStackableItem |
| `-2` | `case -2` → string-pool unavailable | DropPickUpItemUnavailable |
| `-3` | `case -3` → string-pool game-file-damaged | DropPickUpGameFileDamaged |
| `-1` (`default`) | `default` → string-pool inventory-full | DropPickUpInventoryFull |

Matches the structs' baked inner bytes (`-2/-1/-3/0/1/2`). No drift across versions.

**OnQuestRecordMessage** (mode 1): reads `Decode2` (questId) → `Decode1` (inner byte), then:

| inner disc | branch in decompile | Atlas arm |
|---|---|---|
| `0` | `if (!v#)` → RemoveQuest path, no extra read | ForfeitQuestRecord |
| `1` | `==1` → `DecodeStr` (info) → SetQuest | UpdateQuestRecord |
| `2` | `==2` → `DecodeBuffer(...,8)` (FILETIME completedAt) | CompleteQuestRecord |

Matches the quest family inner bytes (`0/1/2`). No drift across versions.

Inner-handler delegate addresses (same as the mode-0/mode-1 rows in §2): OnDropPickUpMessage
v83 `0xA20AD9` / v84 `0xA6BEEF` / v87 `0xAB818C` / v95 `0x9FE190` / jms `0xB07A01`;
OnQuestRecordMessage v83 `0xA20F4C` / v84 `0xA6C362` / v87 `0xAB85D2` / v95 `0xA03920` /
jms `0xB07E49`.

### 4. v84 per-arm semantic confirmation (Step 4 — NOT folded from v83/v95)

Each v84 `sub_` was decompiled and its read order matched the v95-named arm at the **same case
index**. The key check — case 4 SP — confirmed:

- case 4 `sub_A6CEFA` = SP: reads `Decode2` (short jobId, with the `v1/100==22 || v1==2001`
  job-id check) then `Decode1` (byte amount) → exactly v95 `OnIncSPMessage` (`short jobId + byte
  amount`). This proves v84 SP-at-4 directly.
- case 3 `sub_A6CFD7` = EXP (`Decode1` flag + multi-`Decode4` exp breakdown) = v95 `OnIncEXPMessage`.
- case 5 `sub_A6D63F` = fame (`Decode4`, `>=0` gained/lost) = v95 `OnIncPOPMessage`.
- case 6 `sub_A6D705` = meso (`Decode4`, `<=0`, calls CheckQuestCompleteByMeso) = v95 `OnIncMoneyMessage`.
- case 7 `sub_A6D7DB` = GP (`Decode4`, `<=0`) = v95 `OnIncGPMessage`.
- case 8 `sub_A6D8A1` = buff (`Decode4` itemId) = v95 `OnGiveBuffMessage`.
- case 2 `sub_A6CB31` = cash-item-expire (`Decode4` itemId) = v95 `OnCashItemExpireMessage`.
- case 9 `sub_A6CBD7` = general-item-expire (`Decode1` count loop, `Decode4` itemId) = v95.
- case 0xA `sub_A6CEAD` = system-message (`DecodeStr`) = v95 `OnSystemMessage`.
- case 0xB `sub_A6CA40` = quest-record-ex (`Decode2` questId, `DecodeStr` info) = v95.
- case 0xC `sub_A6CCB3` = item-protect-expire (`Decode1` count loop, `Decode4` itemId) = v95.
- case 0xD `sub_A6CD8F` = item-expire-replace (`Decode1` count loop, `DecodeStr`) = v95.
- case 0xE `sub_A6CDF3` = skill-expire (`Decode1` count loop, `Decode4` skillId) = v95.
- cases 0/1 (fan-out) `sub_A6BEEF` / `sub_A6C362` confirmed identical inner discriminators (§3).

Conclusion: v84 mode table = v83 with SP inserted at 4, every arm from fame up shifted +1.
Verified per-arm, not assumed.

### 5. jms `sub_B0931C` (outer mode 0xF) — ESCALATED (Step 5 STOP-AND-ASK GATE)

Decompiled at `0xB0931C` (jms_v185, port 13339). Full read order:

```
v3 = CInPacket::Decode4(iPacket)            // a single 4-byte int
StringPool::GetString(..., 0x15E3)          // string-pool id 5603
ZXString::Format(&s, <fmt>, v3)             // format that one int into the message
sub_4A586D(&s, 6u)                          // screen/chat message add, font arg = 6
```

So mode 0xF reads exactly **one `Decode4` int** and displays a single formatted string-pool
message (id 0x15E3). It does NOT match any DROP_PICK_UP / QUEST_RECORD inner arm (those are
mode 0/1), and although the *read shape* (one Decode4) coincides with several singleton arms
(CashItemExpire/IncreaseFame/IncreaseMeso/IncreaseGuildPoint/GiveBuff), those already occupy
their own outer modes (2/5/6/7/8) in the jms switch. Mode 0xF is a **new outer mode that exists
in no GMS version** (v83=0–0xD, v84/87/95=0–0xE) and has **no Atlas `StatusMessage*` struct and
no operations key** (the 24 Atlas arms span exactly modes 0–0xE; see status_message_body.go).

**Disposition: RESOLVED.** Human decision: implement as a new jms-only arm `StatusMessageJMSCounterNotice`
/ operations key `JMS_COUNTER_NOTICE` / jms mode 15 (0xF) / delegate `0xB0931C` / wire `[mode byte][int32 amount]` /
semantics: single int formatted into localized StringPool 5603, displayed as chat-type-6 line.
Message text is runtime-encrypted and intentionally not asserted in the name. The 15 shared arms
(modes 0–0xE) are fully grounded above; jms ✅ for those does not depend on this arm.
