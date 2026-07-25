# AP/SP Reset Cash Items — Design

Task: task-126-ap-sp-reset-items
Status: Approved PRD → this document covers architecture, alternatives, and tradeoffs.
Inputs: `docs/tasks/task-126-ap-sp-reset-items/prd.md`, code exploration of atlas-channel,
atlas-character, atlas-skills, atlas-saga-orchestrator, libs/atlas-saga, libs/atlas-packet,
libs/atlas-constants, seed templates, and the Cosmic reference checkout (repo-external).

---

## 1. Summary

AP Reset (5050000) and SP Reset (5050001–5050004) are implemented as:

1. A new serverbound sub-body codec `ItemUsePointReset` in `libs/atlas-packet/cash/serverbound/`
   (two int32s, **To then From**, plus a trailing updateTime on non-GMS≥95 versions — IDA-verified
   per version).
2. A new arm in atlas-channel's `CharacterCashItemUseHandleFunc` that decodes the sub-body,
   performs **server-side pre-validation** (every §4.2–§4.4 PRD rule except the job pool-minimum
   check, which stays with the table owner — §7), and on pass creates a two-step saga.
3. A new saga type `point_reset` with step order **[`destroy_asset`, `transfer_ap`|`transfer_sp`]**
   and reverse-walk compensation (destroy → re-award) modeled on the existing PetEvolution
   compensator.
4. A new `TRANSFER_AP` command in atlas-character and a new `TRANSFER_SP` command in atlas-skills,
   each validating authoritatively inside one DB transaction and emitting success/error status
   events keyed by `transactionId`.
5. Pink-text + enable-actions feedback rendered by atlas-channel — directly on pre-validation
   failure, and via a new `point_reset` branch in the saga-failed consumer for the rare
   mid-saga failure.

No new REST endpoints. **No schema change** (see §2.1 — the PRD's proposed column already exists).

---

## 2. Corrections to PRD assumptions (discovered during exploration)

These findings supersede the corresponding PRD statements; each is a deliberate design decision.

### 2.1 `hpMpApUsed` already exists — no new column

`services/atlas-character/atlas.com/character/character/entity.go:49` already has
`HpMpUsed int` → column `hpmp_used` (not null, default 0), and the distribute-AP HP/MP arms
already increment it (`character/processor.go:874,885`). This is exactly the PRD §6 counter
under a different name. **Decision: reuse `hpMpUsed` as-is.** No migration, no baseline
publish/restore concern, and the FR-6 gate becomes `c.HpMpUsed() >= 1`.

Two consequences folded into scope:
- **Bug fix:** `modelBuilder.Build()` (`character/model.go:400-430`) omits `hpMpUsed` from the
  returned Model even though `CloneModel` sets it on the builder. Any clone→build cycle silently
  zeroes the counter. This must be fixed as part of this task (it would corrupt the FR-6 gate).
- The existing increment behavior already satisfies the PRD acceptance criterion "increments on
  AP assigned into HP/MP (normal distribution)". Only the AP-reset increment/decrement paths are new.

### 2.2 The primary-stat cap the PRD references does not exist yet

PRD FR-9 says "Atlas uses its existing stat-cap validation in the distribute-AP path" — there is
no such validation: `RequestDistributeAp` and the administrator setters apply no floor or cap
(`character/administrator.go:191-221`). The reference behavior (Cosmic
`AbstractCharacterObject.assignStrDexIntLuk`) enforces floor 4 and cap `MAX_AP`; the local
reference config sets `MAX_AP: 32767` (`config.yaml:305`). **Decision:** the transfer command
enforces floor 4 / cap 32767 on primary stats and cap 30000 on MaxHP/MaxMP (Cosmic
`assignHP`/`assignMP` reject at `maxhp >= 30000`). Constants live with the policy tables (§7).
We do NOT retrofit caps onto the existing distribute path in this task.

### 2.3 Template wiring: sub-body needs nothing; the parent handler is missing on most versions

Sub-body dispatch is purely server-side by item classification — no template entry exists for
pet-consumable/chalkboard/field-effect sub-bodies and none is needed for point-reset. **But**
`CharacterCashItemUseHandle` itself is only wired in the gms_83 and gms_84 seed templates:

| version | wired? | USE_CASH_ITEM opcode (registry `status.json`) |
|---|---|---|
| gms_v83 | yes (0x4F) | 0x4F |
| gms_v84 | yes (0x4F) | 0x4F |
| gms_v87 | **no** | 0x52 |
| gms_v92 | **no** | *unknown — not in registry* |
| gms_v95 | **no** | 0x55 |
| jms_v185 | **no** | 0x47 |

**Decision:** add `CharacterCashItemUseHandle` (with `LoggedInValidator`, per the
silently-dropped-handler gotcha) to the gms_87, gms_95, and jms_185 seed templates at the
registry opcodes, patch live tenant configs (existing tenants do not re-seed), and **park
gms_v92 explicitly** (opcode unverified, no v92 IDB — v92 mount-food precedent). Side effect to
verify at implementation time: wiring the parent handler makes the *other* existing arms
(pet consumable, chalkboard, field effect) reachable on v87/v95/jms for the first time; the
per-version IDA audit of `CWvsContext::SendConsumeCashItemUseRequest` (§5.2) verifies the shared
prefix and trailing-updateTime layout those arms depend on.

### 2.4 `GetCashSlotItemType` maps 5050002–5050004 to type 23, not 24

The existing mapping (`character_cash_item_use.go:140-148`) returns 24 only for `itemId%10 == 1`
(i.e. 5050001); 5050002–5050004 fall through to 23. The PRD keeps the mapping as-is and decides
AP-vs-SP by item id. **Decision:** the new handler arm matches `it == 23 || it == 24` and
branches solely on item id (5050000 → AP; 5050001–5050004 → SP; any other 505x id → existing
warn fall-through). The 23/24 distinction is never used for dispatch.

### 2.5 Skill-macro updates do not reach an online client today

`STATUS_EVENT_TOPIC_SKILL_MACRO` has no consumer anywhere in the repo; macros are only sent at
login (`atlas-channel/.../kafka/consumer/session/consumer.go:321-335`). FR-18 requires the
client to reflect macro cleanup immediately. **Decision:** add a new atlas-channel consumer for
the macro status topic that re-reads macros and writes `CharacterSkillMacroWriter` to the
session — the same packet the login path uses. This benefits any future macro mutation too.

### 2.6 atlas-skills has no game-data client

atlas-skills trusts caller-supplied levels and has no atlas-data dependency. Rather than adding
one, the game-data max level for the target skill (tiers 1–3) is computed in atlas-channel
(`data/skill` client — max level = `len(Model.Effects())`) and carried in the saga payload.
atlas-skills applies: effective cap = character's `masterLevel` row when the target is 4th-job
(per `job.IsFourthJob` of the skill's job prefix), else the payload's `targetMaxLevel`.
atlas-channel is a trusted server-side caller, so this does not violate "never trust the client".
Tradeoff vs. a new data client in atlas-skills: one fewer service dependency and no new REST
fan-out on the hot path, at the cost of a cap value living in the payload; re-validation of
everything state-derived (levels, master level, tier, job tree) still happens inside atlas-skills.

---

## 3. Saga shape — approaches considered

The PRD invariant: *the item is destroyed iff the transfer was applied* (both directions), and
validation failures must never consume the item.

**A. `[transfer, destroy]` + custom inverse-transfer compensation.**
Common failures (validation) fail step 1 with nothing mutated — clean. But if `destroy` fails
after the transfer succeeded (item dropped/moved/double-used in the race window), compensation
must revert the transfer, and the inverse is **not value-symmetric for HP/MP** (gain into HP is
+20 for a warrior; taking it back out is −54). Exact revert requires capturing applied deltas in
step state — new machinery the compensator doesn't have (the cosmetic-change compensators
document exactly this "payload didn't capture the old value" limitation). Also, a double-use
race applies the transfer twice before the second destroy fails.

**B. `[destroy, transfer]` + reverse-walk compensation (destroy → re-award) + full channel
pre-validation. (Chosen.)**
- Destroy-first makes double-use safe: the second saga's destroy fails at step 1 with nothing
  mutated.
- If the transfer step fails (rare race — pre-validation already passed in channel), the
  existing compensation *pattern* applies: PetEvolution's reverse-walk already inverts
  `DestroyAsset → RequestCreateItem` (`saga/compensator.go:1105-1140`); `point_reset` registers
  its own saga-type-level reverse-walk the same way. No inexact inverse math anywhere.
- The common failure mode (player attempts an invalid reset) never creates a saga at all:
  atlas-channel pre-validates the §4.2–§4.4 rules (all but the pool-minimum check, §7) and
  answers with pink text + enable-actions immediately. Saga-level validation failure is reduced
  to genuine races plus the pool-minimum rejection class.
- Cost: in the rare compensated case the item is destroyed and re-created (brief window where
  the transfer hasn't happened and the item is gone — eventually consistent; the re-award cannot
  fail for capacity since the destroy just freed the slot). Re-created item fidelity (expiration
  attributes) follows the existing PetEvolution inversion behavior; acceptable for a consumable.

**C. SP transfer as two generic `REQUEST_UPDATE` skill steps.**
Rejected outright: the orchestrator correlates step completion by `transactionId + EventKind`
only — two `UPDATED` events under one transaction are ambiguous — and two independent commands
are not atomic across the two skill rows + macros (and the `ExecuteTransaction` no-op bug makes
"two separate writes" genuinely non-atomic today). A purpose-built `TRANSFER_SP` command doing
both rows + macros in one transaction is required regardless of saga shape.

**Decision: B.** Step order `[destroy_asset, transfer_ap|transfer_sp]`, new
`SagaType: point_reset`, `InitiatedBy: "CASH_ITEM_USE"`, with a `point_reset` reverse-walk case
in `CompensateFailedStep` that inverts the completed `destroy_asset` via the existing
re-award/`RequestCreateItem` mechanics.

---

## 4. Component design

### 4.1 libs/atlas-constants (additions)

- `job.Advancement(jobId Id) int` — returns the job-advancement tier 0–4:
  `IsBeginner → 0`; `jobId%100 == 0 → 1`; else `2 + jobId%10`. Verified against the id scheme
  for Explorers (100→1, 110/120/130→2, 111→3, 112→4), Cygnus (1100→1 … 1112→4), and Aran
  (2100→1 … 2112→4). **Evan (2001, 2200–2218) is explicitly excluded** — see §9.4. Table-driven
  tests cover all supported lines.
- `skill.IsPointResetExcluded(skillId Id) bool` (name finalized at plan time) — the FR-13
  exclusion set, taken from the Cosmic gates (`AssignSPProcessor.canSPAssign:43-59`,
  `GameConstants.isPqSkill:547-548`, `isGMSkills:555-556`):
  - Aran hidden combo skills: 21110007, 21110008, 21120009, 21120010.
  - GM skills: 9001000–9101008 and 8001000–8001001.
  - PQ skills: `(20000014–20000018) || 10000013 || 20001013 || (id%10000000 in 1009–1011) ||
    id%10000000 == 1020`.
- Skill tier of a skill id = `job.Advancement(job.IdFromSkillId(skillId))`; tier 0 (beginner
  prefix) is excluded as both source and target (FR-13) — no new helper needed beyond
  `Advancement`.
- Job-tree membership: `job.Is(characterJobId, job.IdFromSkillId(skillId))` — the existing
  branch-arithmetic check (`job/model.go:41`); no curated skill lists (those are incomplete by
  design and would false-reject).

### 4.2 libs/atlas-packet — `ItemUsePointReset`

New `cash/serverbound/item_use_point_reset.go`, following the `ItemUseFieldEffect` pattern
(`item_use_field_effect.go:12-49`):

```go
type ItemUsePointReset struct {
    to              uint32 // wire order: To first, then From (Cosmic UseCashItemHandler:178-183, :224-225)
    from            uint32
    updateTime      uint32 // trailing; only when !updateTimeFirst
    updateTimeFirst bool   // injected config, not a wire field
}
func NewItemUsePointReset(updateTimeFirst bool) *ItemUsePointReset
```

`Decode`: read `to` (uint32), `from` (uint32); if `!updateTimeFirst`, read trailing
`updateTime`. This resolves the standing TODO at `character_cash_item_use.go:108` for this arm.
The same body shape is used for AP (stat flags) and SP (skill ids); interpretation happens in
the handler.

**Verification discipline (per `docs/packets/audits/VERIFYING_A_PACKET.md` §9):**
- The exact read order (To/From, trailing updateTime presence) MUST be IDA-verified per version
  against `CWvsContext::SendConsumeCashItemUseRequest` before the codec is finalized — the
  Cosmic order is a hypothesis, not evidence. `USE_CASH_ITEM` is currently *incomplete in every
  version* (no audit report exists anywhere).
- Required artifacts per wired version: `// packet-audit:verify` byte-fixture tests (both
  round-trip table and exact-bytes fixtures, per the `shop_operation_buy_test.go` style), audit
  REPORT json/md under `docs/packets/audits/<version>/`, and a new `candidatesFromFName` case for
  `CWvsContext::SendConsumeCashItemUseRequest` in the packet-audit tool's `cmd/run.go`.
- gms_v92 cannot be verified (no IDB) → parked, not guessed (PRD open question 1 resolution).

### 4.3 atlas-channel — handler arm, pre-validation, feedback

**Handler arm** (in `CharacterCashItemUseHandleFunc`, above the fall-through warn):

1. Match `it == CashSlotItemType(23) || it == CashSlotItemType(24)`; decode
   `NewItemUsePointReset(updateTimeFirst)`.
2. Branch on item id: `5050000` → AP; `5050001–5050004` → SP (tier = `itemId % 10`); any other
   id → warn + enable-actions (impossible from a legit client).
3. Dead check (FR-4): channel character model `Hp() == 0` → enable-actions only, no pink text
   (Cosmic parity, `UseCashItemHandler:172-175`), item untouched.
4. **AP pre-validation** — map wire flags 64/128/256/512/2048/8192 to the existing ability
   constants (`STRENGTH`/`DEXTERITY`/`INTELLIGENCE`/`LUCK`/`HP`/`MP`,
   `atlas-character character/processor.go:45-50` naming); any other flag → warn +
   enable-actions (Cosmic's default arm sends the empty stat update only). Then evaluate the
   §4.2/§4.3 rules that don't need the numeric policy tables — stat floor ≥ 5, primary/pool
   target caps, and the `HpMpUsed ≥ 1` gate — against the channel character model (stats, job,
   level are already on it) plus `HpMpUsed`, which must be **added to the atlas-character REST
   model and the channel character model** (small additive REST change). The pool-minimum check
   is atlas-character's alone (§7).
5. **SP pre-validation** — the channel character model already carries `skills []skill.Model`
   (`character/model.go:57`), giving current level and master level; game-data max level via the
   existing `data/skill` client (`len(Effects())`). Checks: both skills in job tree
   (`job.Is`), exclusion list, tier-0 exclusion, target tier == item tier, source tier ≤ item
   tier, source level > 0, target below effective cap (master level for 4th job, else game-data
   max).
6. On any pre-validation failure: pink text via `WorldMessagePinkTextBody` +
   `chatpkt.WorldMessageWriter` and enable-actions via the empty
   `statpkt.StatChangedWriter` announce (both `IfPresentByCharacterId` — the exact pairing the
   consumable error consumer uses, `kafka/consumer/consumable/consumer.go:57-80`). Item
   untouched, no saga. Message wording per Cosmic (§6).
7. On pass: create the saga:

```go
saga.Saga{
  TransactionId: uuid.New(),
  SagaType:      saga.PointReset,          // new
  InitiatedBy:   "CASH_ITEM_USE",
  Steps: []saga.Step{
    {StepId: "consume_point_reset_item", Action: saga.DestroyAsset,
     Payload: saga.DestroyAssetPayload{CharacterId, TemplateId: itemId, Quantity: 1}},
    {StepId: "transfer_point", Action: saga.TransferAp /* or TransferSp */,
     Payload: /* §4.4 */},
  },
}
```

**Failure feedback (saga path):** new `SagaTypePointReset` branch in `handleFailedEvent`
(`kafka/consumer/saga/consumer.go:78-130`), mapping the failed event's `ErrorCode` to pink text
(specific message when the code is specific, generic "Couldn't execute AP reset operation."
otherwise) + enable-actions. New saga type + error code constants added to the channel-local
saga message copy.

**Success feedback:** no new plumbing. The transfer's `STAT_CHANGED` event (with
`ExclRequestSent=true`, which the existing provider hardcodes) produces the stat-update packet
that also re-enables actions; skill `UPDATED` events produce `CharacterSkillChangeWriter`
packets via the existing consumer; macro updates flow through the new macro consumer (§2.5).

### 4.4 libs/atlas-saga + atlas-saga-orchestrator

New in `libs/atlas-saga`:
- `TransferAp Action = "transfer_ap"`, `TransferSp Action = "transfer_sp"`; `PointReset Type =
  "point_reset"`.
- Payloads:

```go
type TransferApPayload struct {
    CharacterId uint32     `json:"characterId"`
    WorldId     world.Id   `json:"worldId"`
    ChannelId   channel.Id `json:"channelId"`
    From        string     `json:"from"` // validated ability enum, not wire flags
    To          string     `json:"to"`
}
type TransferSpPayload struct {
    CharacterId    uint32     `json:"characterId"`
    WorldId        world.Id   `json:"worldId"`
    ChannelId      channel.Id `json:"channelId"`
    JobId          uint16     `json:"jobId"`          // character's job, for job-tree/tier re-validation
    FromSkillId    uint32     `json:"fromSkillId"`
    ToSkillId      uint32     `json:"toSkillId"`
    ItemTier       byte       `json:"itemTier"`       // 1–4, for authoritative re-validation
    TargetMaxLevel byte       `json:"targetMaxLevel"` // game-data max, used for non-4th-job cap
}
```

- Mandatory `Step.UnmarshalJSON` cases in `unmarshal.go` for both actions (the silent
  `map[string]any` default would otherwise break the orchestrator's type asserts).
- Channel's curated alias block (`atlas-channel saga/model.go`) gains the new actions/payloads/type.

New in atlas-saga-orchestrator:
- `GetHandler` cases → `handleTransferAp` / `handleTransferSp`, emitting the service commands
  (§4.5, §4.6) with the saga's `transactionId` — async actions, no immediate `StepCompleted`.
- `acceptanceTable` entries: `TransferAp → {EventKindCharacterStatChanged}`; `TransferSp →
  {EventKindSkillSpTransferred}` (new kind, `"skill.sp_transferred"`). The existing coverage
  test enforces these entries.
- Failure path: the character consumer handles the character `ERROR` status event (existing
  envelope, new error type strings) → `StepCompleted(transactionId, false)`; the skill consumer
  gains the same for a new skills `ERROR` status event type. The machine-readable reason from
  the service error event is threaded into the saga-failed event's `ErrorCode` (the
  transport/party_quest services establish this custom-error-code precedent; exact plumbing is a
  plan-phase detail).
- Compensation: `point_reset` saga-type reverse-walk case in `CompensateFailedStep`, inverting
  the completed `DestroyAsset` step via the existing re-award mechanics (PetEvolution's
  `DestroyAsset → RequestCreateItem` inversion is the template).

### 4.5 atlas-character — `TRANSFER_AP`

Modeled on `REBALANCE_AP` end-to-end (command const + body in `kafka/message/character/kafka.go`,
handler in `kafka/consumer/character/consumer.go`, buffered `message.Emit` processor method —
NOT the older inline-producer idiom of `RequestDistributeAp`).

- `CommandTransferAp = "TRANSFER_AP"`, body `{ChannelId, From, To string}` (ability enum).
- `TransferApAndEmit(transactionId, worldId, channelId, characterId, from, to)` →
  `TransferAp(mb)`:
  1. Load character; resolve the job policy row (§7).
  2. Validate source: primary stat ≥ 5; or pool: `HpMpUsed() >= 1` and
     `MaxHp - takeHp >= minHp(job, level)` (resp. MP).
  3. Validate target: primary stat + 1 ≤ 32767; or pool: `MaxHp < 30000` (resp. MP).
     Any failure → emit character `ERROR` status event `{ErrorType, Detail}` with the
     transactionId; **nothing mutated** (validate-then-apply — deliberately avoids Cosmic's bug
     where a target-side failure leaks the source decrement, `addStat` returning false after
     `APResetAction` already applied the source arm).
  4. Apply in ONE `dynamicUpdate`: source dec (primary −1, or MaxHp −takeHp with current HP
     floored at 1 / MaxMp −takeMp with current MP floored at 0, and `HpMpUsed −1`); target inc
     (primary +1, or MaxHp/MaxMp + deterministic gain and `HpMpUsed +1`). `remainingAp`
     untouched (FR-11). From==To processed like any other pair (FR-12).
  5. Emit `STAT_CHANGED` (transactionId, `ExclRequestSent=true`, all affected `stat.Type`s —
     including `TypeHp`/`TypeMp` when current values moved).
- New error type strings on the existing character `ERROR` status event: `STAT_AT_MINIMUM`,
  `STAT_AT_MAXIMUM`, `INSUFFICIENT_HPMP_AP_USED`, `POOL_BELOW_JOB_MINIMUM`, `INVALID_TARGET`
  (bad ability value — unreachable via channel but authoritative), each with a `Detail` field
  naming the stat.
- REST: expose `hpMpUsed` on the character REST model (read-only attribute) for channel
  pre-validation.
- **Gain values are the PRD §4.3 deterministic tables — NOT `getMaxHpGrowth`/`getMaxMpGrowth`**,
  which set absolute values, add passive-skill bonuses, and carry level-up gates (`HpMpUsed >
  9999`) that don't apply to the reset path. The reset path adds a flat delta and caps at 30000.

### 4.6 atlas-skills — `TRANSFER_SP`

- `CommandTypeTransferSp = "TRANSFER_SP"`, body `{FromSkillId, ToSkillId, ItemTier,
  TargetMaxLevel}` on `COMMAND_TOPIC_SKILL`.
- Processor `TransferSp` (buffered emit), all inside ONE gorm transaction spanning both packages
  (skill processor `WithTransaction` exists at `skill/processor.go:86`; **add the same method to
  the macro processor**, which lacks it — this satisfies the NFR that multi-row mutations be
  genuinely atomic despite the known `ExecuteTransaction` no-op bug, because it is one real tx
  handle threaded through):
  1. Re-validate authoritatively: job tree (`job.Is` vs. the character's `JobId`, which the
     payload carries — atlas-skills doesn't store job, and the payload-carried value rests on
     the same trusted-server-caller argument as `TargetMaxLevel`), exclusion list, tiers vs.
     `ItemTier`, source level > 0, target below effective cap (4th-job → own `masterLevel` row;
     else `TargetMaxLevel`).
     Failure → skills `ERROR` status event (new type) with reason `SKILL_AT_ZERO` /
     `SKILL_AT_CAP` / `WRONG_TIER` / `INVALID_TARGET` + detail; nothing mutated.
  2. Apply: source level −1, target level +1 (master levels untouched, FR-15/16).
  3. If source hits 0: clear it from any macro slot (skillId1/2/3 → 0) and persist macros in the
     same tx (FR-18).
  4. Emit: one new `SP_TRANSFERRED` status event (transactionId — the saga completion signal),
     two standard `UPDATED` skill events (one per skill — drives the existing
     `CharacterSkillChangeWriter` path in atlas-channel), and the macro `UPDATED` event when
     macros changed (drives the new macro consumer). The acceptance table gates the orchestrator
     to `SP_TRANSFERRED` only, so the extra `UPDATED` events cannot double-complete the step.

### 4.7 Seed templates + live config

- Add `CharacterCashItemUseHandle` + `LoggedInValidator` rows to `template_gms_87_1.json`
  (0x52), `template_gms_95_1.json` (0x55), `template_jms_185_1.json` (0x47).
- Document + apply the live-tenant config PATCH for existing tenants (channel restart required —
  handlers don't hot-reload), per the "new opcodes missing from live tenant config" gotcha.
- gms_v92: parked with an explicit note (no IDB to verify the opcode or body).

---

## 5. Data flow

### Success (AP example)

```
client USE_CASH_ITEM ──► atlas-channel handler
  decode prefix + ItemUsePointReset ► slot/item check ► dead check ► flag→ability map
  ► pre-validate §4.2/§4.3 ► saga.Create(point_reset)
       step 1 destroy_asset ──► inventory service ► destroyed ► StepCompleted(true)
       step 2 transfer_ap ────► atlas-character TRANSFER_AP
             validate ► one dynamicUpdate ► STAT_CHANGED(txn, exclRequestSent=true)
                    ├─► orchestrator character consumer ► StepCompleted(true) ► saga completed
                    └─► atlas-channel stat consumer ► StatChangedWriter (updates + re-enables client)
```

SP differs only in step 2 (`TRANSFER_SP` → `SP_TRANSFERRED` completes the step; `UPDATED` ×2 →
skill-change packets; macro `UPDATED` → macro packet via the new consumer).

### Failure paths

| Where | What happens | Player sees |
|---|---|---|
| channel pre-validation | no saga, item untouched | pink text + enable-actions |
| step 1 destroy fails (item gone / double-use race) | nothing mutated, saga failed | pink text (generic) + enable-actions |
| step 2 transfer rejected (race, or pool below job minimum — §7) | service emits ERROR ► StepCompleted(false) ► reverse-walk re-awards the item ► saga failed event with ErrorCode | pink text (specific) + enable-actions, item restored |

---

## 6. Error codes → pink text

Codes (PRD §5 set, plus one): `STAT_AT_MINIMUM`, `STAT_AT_MAXIMUM`, `INSUFFICIENT_HPMP_AP_USED`,
`POOL_BELOW_JOB_MINIMUM`, `SKILL_AT_ZERO`, `SKILL_AT_CAP`, `WRONG_TIER`, `INVALID_TARGET`,
`ITEM_UNAVAILABLE` (destroy-step failure). Messages verbatim from Cosmic
(`AssignAPProcessor.APResetAction:486-607`):

- `STAT_AT_MINIMUM` + detail: "You don't have the minimum STR required to swap." (per-stat)
- `INSUFFICIENT_HPMP_AP_USED`: "You don't have enough HPMP stat points to spend on AP Reset."
- `POOL_BELOW_JOB_MINIMUM` + detail: "You don't have the minimum HP pool required to swap." (HP/MP)
- everything else / generic: "Couldn't execute AP reset operation."

SP-side failures have no Cosmic wording (Cosmic fails silently there — deviation 4.6.3); we use
parallel phrasing finalized at plan time (e.g. "That skill's points cannot be moved."). Channel
pre-validation composes the specific message directly; the saga-failed branch maps
`ErrorCode`(+detail where carried) to the same strings.

All impossible-from-a-legit-client rejections log at warn with character id and offending values
(FR-22), both in channel pre-validation and in the services' authoritative checks.

---

## 7. Job policy tables (PRD §4.3)

Location: **atlas-character, new file `character/point_reset.go`** — server policy owned by the
executor, defined once. atlas-channel's pre-validation does NOT mirror the numeric tables:
channel checks the structural rules and the floors/caps/gates it can see cheaply (stat ≥ 5,
`hpMpUsed ≥ 1`, primary/pool target caps), while the pool-minimum check (`minHp`/`minMp`) is
left to atlas-character, whose ERROR event feeds the specific pink text back through the
saga-failed path. Tradeoff: this one legitimate rejection class (pool at job minimum) pays the
destroy-then-compensate saga round-trip instead of failing pre-flight; in exchange the tables
have exactly one owner and cannot drift. If that round-trip proves objectionable, the escape
hatch is moving the tables to `libs/atlas-constants/job` — a mechanical relocation.

Encoding: data, not an if/else chain —

```go
type pointResetPolicy struct{ takeHp, takeMp, gainHp, gainMp uint16 }
// ordered rows: {refs []job.Id (job.IsA semantics), policy}; first match wins; default last.
// min-pool resolver: minHp(jobId, level), minMp(jobId, level) — mult×level+offset rows per PRD §4.3,
// keyed by the finer-grained job-line list (Page/Spearman split, 2nd-job+ splits, Cygnus/Aran rows).
```

Values verbatim from PRD §4.3 (which fixed them from Cosmic `AssignAPProcessor.java:669-986`
under default config). Aran rows are matched by explicit `AranStage*Id` enumeration (no `IsAran`
helper exists; `TypeLegend` also contains Evan). Constants: primary floor 4, primary cap 32767,
pool cap 30000 (§2.2). Table-driven tests assert every row against the PRD tables.

---

## 8. Testing

- **atlas-constants**: `Advancement` across Explorer/Cygnus/Aran ids incl. tier-0; exclusion-list
  predicate.
- **atlas-packet**: round-trip tables (both `updateTimeFirst` values) + IDA-derived byte fixtures
  with `packet-audit:verify` markers per wired version; `matrix --check` clean.
- **atlas-character**: table-driven `TransferAp` validation matrix (every From/To class ×
  floor/cap/gate/min-pool boundaries, From==To), policy-table
  row assertions, `Build()` `hpMpUsed` regression test, ERROR-event emission, STAT_CHANGED
  contents.
- **atlas-skills**: `TransferSp` matrix (tier/job-tree/exclusions/caps incl. 4th-job master-level
  cap), macro cleanup (each slot, multi-slot, no-macro), single-transaction atomicity (failure
  mid-way leaves both rows + macros untouched), event set emitted.
- **atlas-saga-orchestrator**: acceptance-table coverage test entries, handler dispatch,
  `point_reset` reverse-walk compensation (transfer failure → item re-award), failed-event
  ErrorCode threading.
- **atlas-channel**: pre-validation branch tests (Builder-pattern setup — no test-helper files),
  flag mapping, item-id dispatch incl. the 5050002→type-23 quirk, saga assembly, failed-event
  pink-text branch.
- Saga integration criteria from PRD §10 (success, each rejection class, compensation) are
  exercised via the orchestrator + service unit layers; end-to-end on a live tenant is the
  acceptance pass.

Build gates per CLAUDE.md: `go test -race`, `go vet`, `go build` per changed module,
`docker buildx bake` for atlas-channel, atlas-character, atlas-skills, atlas-saga-orchestrator
(+ any lib-touch rebuilds), `tools/redis-key-guard.sh`.

---

## 9. Resolved PRD open questions

1. **Per-version body layout** — resolved by mandatory IDA verification of
   `CWvsContext::SendConsumeCashItemUseRequest` per version before finalizing the codec; v92
   parked (no IDB). The trailing-updateTime hypothesis (v83 TODO) and the Cosmic To/From order
   are treated as unverified until the audit lands.
2. **4th-job target cap** — master level (this design), enforced in atlas-skills from its own
   `masterLevel` row; flipping to literal maxLevel is a one-line change in the effective-cap
   selection.
3. **Pink-text delivery** — existing `WorldMessagePinkTextBody` writer, sent directly by
   atlas-channel (pre-validation failures) and from the saga-failed consumer branch (saga
   failures). The `send_message` saga action is not used — feedback is not part of the
   transaction and must also fire when no saga exists.
4. **Cygnus/Aran coverage** — full stage ids exist in `libs/atlas-constants/job/constants.go`
   (`DawnWarriorStage4Id=1112`, `AranStage4Id=2112`, etc.); `Advancement` and the policy tables
   handle them. **Evan is deliberately unsupported for SP Reset** (rejected `WRONG_TIER`, warn
   log): the tier-scoped items don't map onto Evan's 10-stage/skill-book system, and defining a
   mapping would be invention. Documented as deviation; only v84+ tenants can even roll Evan.

## 10. Deviations & known limitations (beyond PRD §4.6)

- Evan SP Reset rejected (see §9.4).
- gms_v92 wiring parked (no IDB) — item is inert there, as today.
- The compensated-failure window (§3.B) is eventually consistent: item briefly absent before
  re-award if a race kills the transfer step.
- Primary-stat cap 32767 / pool cap 30000 / floor 4 are fixed constants (reference-config
  parity), not tenant-configurable — no configuration surface exists for them today and none is
  added.
