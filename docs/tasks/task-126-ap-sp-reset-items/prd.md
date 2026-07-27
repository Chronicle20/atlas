# AP/SP Reset Cash Items — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-07-02
---

## 1. Overview

AP Reset (item 5050000) and SP Reset (items 5050001–5050004) are cash items that let a player
move a single already-spent point: AP Reset moves one ability point from one stat (STR, DEX,
INT, LUK, HP, or MP) to another; each SP Reset moves one skill point out of a skill and into
another skill of the item's job tier.

Atlas already classifies these items — `item.ClassificationPointReset` (505) maps to
`CashSlotItemType` 23 (AP Reset) / 24 (SP Reset) in
`services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go` — but no
handler arm exists for those types. Today the packet falls through to a warning log and the
item does nothing. This task implements the feature end-to-end: serverbound packet decode,
server-side validation, a dedicated saga that applies the point transfer and consumes the item,
and the client-facing stat/skill/failure feedback.

Reference behavior is Cosmic (`UseCashItemHandler.java:171-231`,
`AssignAPProcessor.APResetAction`, `AssignSPProcessor.canSPAssign`), with deliberate deviations
listed in §4.6. Cosmic source citations in this document are from the local checkout at
`Cosmic/src/main/java/` (repo-external reference).

## 2. Goals

Primary goals:
- Players can use AP Reset (5050000) to move 1 AP between STR/DEX/INT/LUK/HP/MP with
  Cosmic-parity validation (stat floor 4, job-based HP/MP pool rules, `hpMpApUsed` gate).
- Players can use SP Reset (5050001–5050004) to move 1 SP, with job-tier enforcement matching
  the item descriptions (see §4.4).
- The item is consumed if and only if the transfer was applied.
- Validation failures give the player pink-text chat feedback and re-enable client actions.
- Works on all supported tenant versions (gms v83, v84, v87, v92, v95; jms v185), with
  per-version packet verification per `docs/packets/audits/VERIFYING_A_PACKET.md`.

Non-goals:
- Cash-shop purchase/gifting of these items (generic cash-shop flow already covers acquisition).
- UI (atlas-ui) changes.
- Other Cash/505x/506x item families (item tag, sealing lock, incubator, etc.).
- Cosmic's autoban/disconnect responses to tampered packets (we reject + log instead).
- Retroactive backfill of `hpMpApUsed` for existing characters (see §6).

## 3. User Stories

- As a player, I want to use an AP Reset to move a misplaced ability point (e.g. INT → STR on
  a warrior) so that I can correct my build without remaking the character.
- As a player, I want to use an AP Reset to drain HP/MP-invested AP back into a main stat,
  limited to the number of points I actually put into HP/MP, so that stat washing works the
  way it does on the reference server.
- As a player, I want to use the SP Reset matching a job tier to move a skill point into a
  skill of that tier so that I can fix skill-build mistakes.
- As a player, I want a clear pink-text message when a reset is not allowed (stat at minimum,
  skill maxed, wrong job tier) and my item NOT consumed, so failed attempts don't cost me NX.
- As an operator, I want tampered or malformed requests rejected server-side and logged, so
  packet-edited resets cannot corrupt character state.

## 4. Functional Requirements

### 4.1 Packet decode & dispatch (atlas-channel, libs/atlas-packet)

- FR-1: A new serverbound sub-body codec (working name `ItemUsePointReset`) is added under
  `libs/atlas-packet/cash/serverbound/`, following the existing `item_use_*` codec pattern.
  Per Cosmic the body after the common ItemUse prefix is two int32s, read as **To** then
  **From** (`UseCashItemHandler.java:178-183` SP, `:224-225` AP). The exact per-version byte
  layout (including the v83 trailing `updateTime` noted in the TODO at
  `character_cash_item_use.go:108` and the GMS≥95 updateTime-first prefix) MUST be verified
  against each tenant version's IDA export before implementation; byte-fixture tests are
  required per codec per version.
- FR-2: `CharacterCashItemUseHandleFunc` gains an arm for `CashSlotItemType` 23 and 24. The
  AP-vs-SP branch is decided by item id — `itemId == 5050000` → AP reset, `itemId` in
  5050001–5050004 → SP reset — matching Cosmic's `itemId > ItemId.AP_RESET` split. The
  existing `GetCashSlotItemType` 23/24 mapping is kept as-is (confirmed correct).
- FR-3: The handler verifies (as it already does for all cash items) that the item in the
  claimed cash-inventory slot matches the sent item id; mismatch → reject, log warn, no-op.
- FR-4: A dead character cannot use either item: reject with enable-actions, item not
  consumed (Cosmic `:172-175`).

### 4.2 AP Reset semantics (atlas-character)

Stat identifiers on the wire use the client's stat flag encoding: 64=STR, 128=DEX, 256=INT,
512=LUK, 2048=MaxHP, 8192=MaxMP (Cosmic `AssignAPProcessor.APResetAction:486-607`). Any other
flag value in From or To → reject.

- FR-5: Source STR/DEX/INT/LUK: the current **base** stat must be ≥ 5 before the swap (the
  post-swap floor is 4, matching the item description and Cosmic's `< 5` check). On success
  the stat is decremented by 1.
- FR-6: Source HP (2048): permitted only when the character's `hpMpApUsed` counter is ≥ 1
  (see §6). MaxHP is reduced by the job-based `takeHp` amount and must not drop below the
  job/level minimum `getMinHp` (both tables in §4.3). Current HP is reduced by the same
  amount, floored at 1 (Cosmic default `USE_FIXED_RATIO_HPMP_UPDATE: false`). On success
  `hpMpApUsed` is decremented by 1.
- FR-7: Source MP (8192): same as FR-6 with `takeMp` / `getMinMp` and current MP floored at 0.
- FR-8: There is NO restriction that HP/MP points may only move to MP/HP (Cosmic default
  `USE_ENFORCE_HPMP_SWAP: false`).
- FR-9: Target STR/DEX/INT/LUK: incremented by 1, capped at the configured max stat value
  (Cosmic `MAX_AP`; Atlas uses its existing stat-cap validation in the distribute-AP path —
  target at cap → reject the whole transfer, source untouched).
- FR-10: Target HP/MP: MaxHP/MaxMP increases by the deterministic AP-reset gain for the
  character's job (§4.3 "gain" columns — Cosmic's `calcHpChange`/`calcMpChange` with
  `usedAPReset = true`, which is deterministic under both randomize settings except warrior
  MP where the non-random path gives 3; we use the Cosmic default-config values). On success
  `hpMpApUsed` is incremented by 1.
- FR-11: `remainingAp` (unspent AP) is NOT consumed or granted by an AP reset; the transfer
  moves an already-spent point.
- FR-12: From == To is a valid no-op-shaped request; follow Cosmic: it is processed like any
  other transfer (validations still apply, e.g. STR≥5 for STR→STR). No special-case rejection.

### 4.3 Job-based HP/MP tables (Cosmic parity)

Job classification uses the character's job branch (Cygnus equivalents map to their listed
Explorer branch; Aran per its own rows). Values below are IDA-independent server policy and
are taken verbatim from Cosmic (`AssignAPProcessor.java:669-986`, defaults
`USE_RANDOMIZE_HPMP_GAIN: true`).

Loss when resetting OUT of a pool (`takeHp` / `takeMp`):

| Job branch | HP loss | MP loss |
|---|---|---|
| Warrior / Dawn Warrior / Aran | 54 | 4 |
| Magician / Blaze Wizard | 10 | 31 |
| Bowman / Wind Archer | 20 | 12 |
| Thief / Night Walker | 20 | 12 |
| Pirate / Thunder Breaker | 42 | 16 |
| Beginner / Noblesse (default) | 12 | 8 |

Gain when resetting INTO a pool (AP-reset path, deterministic):

| Job branch | HP gain | MP gain |
|---|---|---|
| Warrior / Dawn Warrior | 20 | 2 |
| Aran | 20 | 2 |
| Magician / Blaze Wizard | 6 | 18 |
| Bowman / Wind Archer | 16 | 10 |
| Thief / Night Walker | 16 | 10 |
| Pirate / Thunder Breaker | 18 | 14 |
| Beginner (default) | 8 | 6 |

Minimum pool after loss (`multiplier × level + offset`), reject if MaxHP/MaxMP would fall
below:

| Job | minHP mult/off | minMP mult/off |
|---|---|---|
| Warrior, Page-line, Spearman-line, DW1, Aran1 | 24 / 118 | 4 / 55 (Warrior, Fighter-line, DW, Aran); 4 / 155 (Page-, Spearman-line) |
| Fighter-line, DW2+, Aran2+ | 24 / 418 | 4 / 55 |
| Magician-line, Blaze Wizard | 10 / 54 | 22 / −1 (base); 22 / 449 (2nd job+) |
| Bowman/Thief base, WA1, NW1 | 20 / 58 | 14 / −15 |
| 2nd-job+ Bowman/Thief lines, WA2+, NW2+ | 20 / 358 | 14 / 135 |
| Pirate base, TB1 | 22 / 38 | 18 / −55 |
| Brawler/Gunslinger lines, TB2+ | 22 / 338 | 18 / 95 |
| Beginner / Noblesse | 12 / 38 | 10 / −5 |

The implementation should encode these as data (job-branch keyed), not a copy of Cosmic's
if/else chain; the design phase decides where they live (atlas-character is the executor).

### 4.4 SP Reset semantics (atlas-skills)

Wire order: target skill id (SPTo) first, then source skill id (SPFrom) (Cosmic `:178-183`).

- FR-13: Common validation for both skills:
  - Both must exist in game data and belong to the character's job tree (job-tree membership
    per the character's current job line, using `libs/atlas-constants` job/skill helpers).
  - Beginner-tier skills (skill tier 0, i.e. skill id job prefix ≤ beginner) are excluded as
    both source and target (their SP is level-derived, not pool-backed).
  - Hidden/system skills (e.g. Aran hidden combo skills) are excluded.
  - GM skills and PQ-only skills are excluded as targets (Cosmic gates these; we reject + log
    rather than autoban).
- FR-14: Tier enforcement per item (deviation from Cosmic, which ignores the tier — per item
  descriptions in Cash/0505.img):
  - Item 505000**N** (N = 1..4) requires the **target** skill to be exactly job-advancement
    tier N of the character's job line.
  - The **source** skill may be any tier ≤ N of the same job line ("1st job SP raised AFTER
    the 2nd job adv. can also be reset"), with tier < N sources permitted unconditionally —
    Atlas does not track when a point was allocated, so the "raised after the advancement"
    qualifier is not enforceable and is deliberately relaxed to "any lower-tier skill".
- FR-15: Source skill current level must be > 0; on success it is decremented by 1. The
  skill's master level (for 4th-job skills) is unchanged.
- FR-16: Target skill current level must be below its cap: `maxLevel` from game data for
  tiers 1–3, and the character's **master level** for 4th-job skills. (Cosmic's reset path
  checks `maxLevel` even for 4th job — `UseCashItemHandler.java:188` — which is inconsistent
  with its own SP-assign path; we use the master-level cap. See Open Questions.)
- FR-17: No unspent SP is consumed or granted; remaining-SP counters are untouched.
- FR-18: If the source skill reaches level 0, any skill-macro slots referencing it are
  cleared and the updated macros are pushed to the client (Cosmic `:192-221`; Atlas macros
  live in atlas-skills' `macro` package).

### 4.5 Transactionality, consumption, and feedback

- FR-19: The flow is a dedicated saga (per interview decision): new saga actions (working
  names `transfer_ap` executed by atlas-character, `transfer_sp` executed by atlas-skills)
  plus the existing `destroy_asset` step for item consumption. Invariant: **the item is
  destroyed iff the transfer step succeeded**. Step ordering / compensation design is a
  design-phase decision, but a failed validation must never consume the item, and a consumed
  item must never leave the character without the applied transfer.
  - Deliberate deviation: Cosmic consumes the SP Reset even when its level-bounds check
    silently fails (`remove` at `:231` is outside the `if` at `:188`); Atlas does not.
- FR-20: On success the client receives the standard stat-update / skill-update packets
  (existing writers used by the distribute-AP/SP and skill-change flows), leaving the client
  unlocked (enable-actions semantics preserved).
- FR-21: On any validation failure the player receives a pink-text chat message describing
  the reason (wording per Cosmic's messages, e.g. "You don't have the minimum STR required
  to swap.") and an enable-actions packet; nothing is mutated and the item is not consumed.
- FR-22: All rejections caused by requests a legitimate client cannot produce (bad stat flag,
  skill outside job tree, beginner/hidden/GM skill, dead character) are logged at warn with
  character id and offending values.

### 4.6 Deviations from Cosmic (summary)

1. SP tier enforcement per item description (Cosmic accepts any job-tree skill for any SP
   Reset tier).
2. 4th-job SP Reset target capped at master level, not maxLevel.
3. No item consumption on the SP silent-failure path.
4. Reject + warn-log instead of autoban/disconnect on tampered requests.
5. Skill-macro storage/cleanup uses Atlas's atlas-skills macro domain.

## 5. API Surface

No new REST endpoints. New surface is Kafka/saga:

- `libs/atlas-saga`: two new `Action` constants and payload structs (names finalized in
  design), e.g.:
  - `TransferAp` — payload: `characterId`, `fromStat`, `toStat` (validated enum, not raw wire
    flags), `transactionId`.
  - `TransferSp` — payload: `characterId`, `fromSkillId`, `toSkillId`, `itemTier` (1–4),
    `transactionId`.
- `atlas-saga-orchestrator`: step executors for both actions, emitting the new commands and
  matching the services' status events for step completion/failure (per the existing
  step-event matching model).
- `atlas-character`: new command type on its character command topic implementing the AP
  transfer (validation from §4.2/§4.3), with success/error status events carrying a
  machine-readable failure reason so atlas-channel can render the pink text.
- `atlas-skills`: new command type implementing the SP transfer (§4.4) plus macro cleanup
  (FR-18), with success/error status events.
- `libs/atlas-packet`: new serverbound codec(s) for the point-reset ItemUse body, wired into
  the version templates for all supported tenants (handler entry requires a validator —
  `LoggedInValidator` — per the socket-handler config rules).
- Existing writers reused for stat/skill updates, pink text, and enable-actions; any missing
  live-tenant config opcodes must be patched per the known
  "new opcodes missing from live tenant config" gotcha.

Error cases (machine-readable reason → pink text): `STAT_AT_MINIMUM`, `STAT_AT_MAXIMUM`,
`INSUFFICIENT_HPMP_AP_USED`, `POOL_BELOW_JOB_MINIMUM`, `SKILL_AT_ZERO`, `SKILL_AT_CAP`,
`WRONG_TIER`, `INVALID_TARGET` (exact enum finalized in design).

## 6. Data Model

- `atlas-character` characters table: new column `hp_mp_ap_used` (integer, not null,
  default 0), tenant-scoped as the table already is. Semantics: incremented when an AP is
  assigned into HP or MP (existing `RequestDistributeAp` HP/MP arms,
  `character/processor.go:865-882`, and the AP-reset target-HP/MP path), decremented when an
  AP reset moves a point out of HP/MP. Never negative.
  - Migration: additive column with default 0; existing characters therefore cannot reset
    out of HP/MP until they invest at least one AP there. Accepted (matches a fresh Cosmic
    DB; no history exists to backfill). Per the known column-order gotcha, the baseline
    publish/restore name-keyed column handling must be confirmed to cover the new column.
- `atlas-skills`: no schema change expected — skill level/master level rows already exist;
  macro cleanup mutates existing macro rows.
- No new tables.

## 7. Service Impact

| Service / lib | Change |
|---|---|
| `libs/atlas-packet` | New `cash/serverbound` point-reset codec + per-version byte-fixture tests; template wiring for all supported versions. |
| `libs/atlas-saga` | Two new actions + payloads + validation. |
| `atlas-channel` | Handler arm for CashSlotItemType 23/24: decode body, dead-check, item/slot check, assemble saga; failure pink-text + enable-actions on error status events. |
| `atlas-saga-orchestrator` | Executors + step-event matching for the two new actions. |
| `atlas-character` | AP transfer command: stat floor/cap, HP/MP tables (§4.3), `hpMpApUsed` column + increment hook in existing distribute-AP HP/MP arms, status events. |
| `atlas-skills` | SP transfer command: job-tree/tier/cap validation, level mutation, macro cleanup, status events. |
| `atlas-data` | None expected (skill max levels already served); confirm during design. |

Seed templates: new handler/writer opcode entries must be added to the version seed templates
AND patched into live tenant configs (existing tenants do not re-seed).

## 8. Non-Functional Requirements

- Multi-tenancy: all commands/events carry tenant context; behavior tables are
  version-independent server policy, packet layouts are per-version.
- Security: never trust client-sent stat flags / skill ids; full server-side validation;
  warn-level logging on impossible requests (FR-22). No autoban system in scope.
- Atomicity: saga with compensation guarantees the consumption invariant (FR-19); note the
  known `ExecuteTransaction` no-op bug — per-service mutations must be genuinely atomic
  where they multi-write (skills: two skill rows + macros).
- Observability: standard structured logs; saga steps visible via existing orchestrator
  tooling.
- Verification discipline: per-version packet work follows
  `docs/packets/audits/VERIFYING_A_PACKET.md` (IDA-derived read order, byte fixtures,
  evidence records); no opcode or layout may be assumed from Cosmic.
- Build gates per CLAUDE.md: `go test -race`, `go vet`, `go build` per changed module,
  `docker buildx bake` per changed service, `tools/redis-key-guard.sh`.

## 9. Open Questions

1. Per-version serverbound body layout (two int32s per Cosmic; v83 trailing updateTime TODO
   at `character_cash_item_use.go:108`; GMS≥95 updateTime-first) — to be resolved by IDA
   verification during design/implementation. If a version's body cannot be verified
   (no IDB), that version's wiring is parked explicitly (v92 precedent).
2. 4th-job target cap: master level (this PRD) vs Cosmic's literal maxLevel — confirm the
   master-level choice at design time; flip is a one-line validation change.
3. Pink-text delivery mechanism: reuse of the existing message/notice writer vs the
   `send_message` saga action — design-phase choice.
4. Whether Cygnus/Aran job lines exist in supported tenant data (v83 has KoC): tier mapping
   must handle them; verify job coverage of `libs/atlas-constants` helpers at design time.

## 10. Acceptance Criteria

- [ ] Using AP Reset 5050000 on each valid From/To pair mutates exactly one point with the
      §4.2/§4.3 rules, consumes the item, and updates the client without relog.
- [ ] AP Reset from a stat at 4, from HP/MP with `hpMpApUsed` = 0, or below the job pool
      minimum is rejected: pink text shown, item retained, no stat change.
- [ ] `hpMpApUsed` increments on AP assigned into HP/MP (both normal distribution and AP
      Reset target) and decrements on AP Reset out of HP/MP; never negative.
- [ ] Using SP Reset tier N moves 1 SP only when the target is tier N of the character's job
      line and the source is tier ≤ N with level > 0; all §4.4 exclusions rejected with pink
      text and item retained.
- [ ] Source skill reaching level 0 clears it from skill macros and the client reflects it.
- [ ] Item consumed iff transfer applied, verified by tests covering the failure paths
      (including a mid-saga failure with compensation).
- [ ] New packet codecs have IDA-verified byte-fixture tests for every supported version they
      are wired into; unverifiable versions are explicitly parked, not guessed.
- [ ] Seed templates updated and a live-tenant config patch documented/applied for the new
      handler opcodes (with validators).
- [ ] All CLAUDE.md build gates pass (`go test -race`, `go vet`, `go build`, `docker buildx
      bake` for changed services, `tools/redis-key-guard.sh`).
