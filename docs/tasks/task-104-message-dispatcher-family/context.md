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
