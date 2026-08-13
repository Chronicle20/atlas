# Aran Combo Counter — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-12
---

## 1. Overview

An Aran wielding a polearm builds a **combo count** as they hit monsters. The
count is drawn on screen by the client, and at combo milestones the character
gains the **Combo Ability** buff (`ARAN_COMBO` temporary stat), which is what
Aran's combo-consuming skills and combo-scaling passives read. Atlas has no
server side for any of this: the client sends its combo-increment request into
a void, the counter never appears, and the `ARAN_COMBO` temporary stat is only
ever produced with a hardcoded value by the static skill-data reader.

Unusually for an attack-derived mechanic, **the client drives the increment**.
`CUserLocal::RequestIncCombo` is invoked from `CMob::OnHit` — not from the
attack packet path — gated client-side on the character owning Combo Ability
(`21000000`, or `20000017` when the job is Legend `2000`) and wielding a
polearm (weapon type 44). It emits a **body-less** packet: verified for v84 in
`docs/packets/registry/gms_v84.yaml:3658` — "RequestIncCombo @0x99f346 pushes
COutPacket(169)=0xa9 (no body, m_bHoldCombo guard)". The server's job is to
re-validate the request against authoritative state, advance the count, decay
it when the player stops hitting things, echo the new count back via
`SHOW_COMBO` (`CUserLocal::OnIncComboResponse`), and keep the `ARAN_COMBO` buff
in sync.

Because the packet has no body, there is nothing to trust in it — every gate
(job, learned skill, equipped weapon, current count) must be evaluated
server-side. This makes the feature small in wire surface and almost entirely
about state management.

### 1.1 Relationship to prior work

Three tasks have circled this and each explicitly excluded it:

- **task-142 (combo attack orbs)** built the *Warrior/Crusader* Combo Attack
  orb system — a different mechanic sharing a word. Its PRD (§`prd.md:39,53`)
  identified `RequestIncCombo`/`SHOW_COMBO` as Aran-only and declared them out
  of scope. It left behind exactly the machinery this task needs: a counter
  stored as a **stat value on a buff**, mutated by the `UPDATE_STAT_VALUE`
  Kafka command (`services/atlas-channel/.../character/buff/processor.go:93` →
  `services/atlas-buffs/.../character/registry.go:360`) with `INCREMENT`
  (cap-clamped) and `SET` operations, plus the attack-pipeline hook shape in
  `services/atlas-channel/.../socket/handler/character_attack_combo.go`.
- **task-166 (Combo Drain)** is the Aran 2nd-job HP-leech buff. Its non-goals
  name "Combo-orb consumption or any Aran combo-counter mechanics"
  (`prd.md:66`).
- **task-187 (version-aware id semantics)** established
  `constants.For(region, major, minor).Skill.Resolve`. Neither `21000000` nor
  `20000017` appears in
  `docs/tasks/task-187-version-aware-id-semantics/audit/divergences.csv`, so
  the Aran combo ids are not on the version-divergent list and
  `tools/skill-job-id-guard.sh` does not bar direct comparison against them.

### 1.2 Correction to the originating backlog entry

The backlog line that spawned this task comes from the missing-features
research corpus (`docs/research/missing-features/skills-and-buffs.md` §7, and
`new-jobs-and-version-delta.md` §5 "Aran Combo Counter"). **That corpus is
untracked in the main repo and therefore absent from this worktree** — quotes
from it in this PRD were read from the main checkout and are reproduced here
so the PRD stays self-contained. The entry is titled
"Aran combo counter (v84+)". **The version anchor is wrong.**
`docs/packets/audits/STATUS.md:726` shows `ARAN_COMBO_COUNTER` present from
**v83** onward — `n-a` for v48/v61/v72/v79, then v83 `0x0A3`, v84 `0x0A9`,
v87 `0x0AD`, v92 `0x0BA`, v95 `0x0BD`, jms185 `0x09D`. Aran is creatable from
v83 (`jobIndex 2` in `template_gms_83_1.json`). v84 is the version that adds
**Evan**, not Aran. This PRD scopes to v83+.

## 2. Goals

Primary goals:

- An Aran (or Legend) with Combo Ability learned and a polearm equipped
  accumulates a server-authoritative combo count as they hit monsters, and the
  count is drawn on their client.
- The count decays server-side when the player stops landing hits, via a
  periodic tick task — no client cooperation required.
- The `ARAN_COMBO` temporary stat reflects the live combo state rather than a
  hardcoded constant, so that downstream consumers read something meaningful.
- All six versions where `ARAN_COMBO_COUNTER` exists (v83, v84, v87, v92, v95,
  jms185) are implemented and their coverage-matrix cells promoted, for both
  `ARAN_COMBO_COUNTER` (serverbound) and `SHOW_COMBO` (clientbound).
- No behavior change for any character who is not an Aran/Legend with the
  combo prerequisites.

Non-goals:

- **Combo-consuming skills.** Combo Smash (`21100004`), Combo Fenrir
  (`21110004`) and Combo Tempest (`21120006`) spending combo is **out of
  scope** — it overlaps task-166's attack-pipeline surface (there is already a
  `// TODO ComboTempest` at
  `services/atlas-channel/.../socket/handler/character_attack_common.go:1051`)
  and would duplicate work in the same functions. This task leaves the count
  spendable but never spends it.
- Combo-scaling damage formulas (Combo Critical `21110000` and friends).
- Aran advancement quests, or any other Aran gap (Body Pressure, Snow Charge —
  see the `TODO(post-task-054)` at
  `services/atlas-data/.../skill/reader.go:475`).
- The `SHOW_COMBO` v79 column. `SHOW_COMBO` exists at v79 (`0x0D2`,
  `STATUS.md:352`) but `ARAN_COMBO_COUNTER` is `n-a` there, so there is no way
  for a v79 client to drive the counter. v79 stays `n-a`/`❌`.
- Evan dragon, Dual Blade, Resistance, Mechanic (siblings in the same research
  doc).

## 3. User Stories

- As an Aran player, I want my combo count to climb on screen as I hit
  monsters with my polearm, so the class reads and feels the way the client
  was built to present it.
- As an Aran player, I want my combo to fall away when I stop fighting, so the
  counter reflects sustained combat rather than an accumulated lifetime total.
- As an Aran player, I want the Combo Ability buff to appear and update as my
  combo crosses its thresholds, so the buff icon and its stat match what the
  counter says.
- As a non-Aran player (or an Aran without a polearm), I want nothing about
  attacking to change, and I want no combo packets sent to me.
- As an operator, I want a client that spams `ARAN_COMBO_COUNTER` — or sends it
  while ineligible — to be rejected server-side without error spam or
  measurable load.

## 4. Version Scope

Six versions are in scope, one seed template each. The serverbound opcodes
below are from `docs/packets/registry/gms_v*.yaml` / `jms_v185.yaml`; the
clientbound from the same registries and `STATUS.md:352`.

| Version | Template | `ARAN_COMBO_COUNTER` | `SHOW_COMBO` | Opcode provenance |
|---|---|---|---|---|
| gms v83 | `template_gms_83_1.json` | `0x0A3` | `0x0E1` | `csv-import` — **unverified** |
| gms v84 | `template_gms_84_1.json` | `0x0A9` | `0x0E6` | serverbound IDA-verified (task-100 cluster-H); clientbound `csv-import` |
| gms v87 | `template_gms_87_1.json` | `0x0AD` | `0x0EF` | `csv-import` — **unverified** |
| gms v92 | `template_gms_92_1.json` | `0x0BA` | `0x103` | `csv-import` — **unverified** |
| gms v95 | `template_gms_95_1.json` | `0x0BD` | `0x101` | `csv-import` — **unverified** |
| jms v185 | `template_jms_185_1.json` | `0x09D` | `0x0EB` | `csv-import` — **unverified** |

Out of scope: v12, v48, v61, v72 (both ops `n-a`), v79 (`ARAN_COMBO_COUNTER`
`n-a`).

**Opcode risk.** Only v84's serverbound opcode has been IDA-verified, and that
verification *moved it* — task-100 found `0xA3` was stale for v84 and the true
value is `0xA9` (`0xA3` was freed for `NEW_YEAR_CARD_REQUEST`). Every other
cell in the table above is `provenance: csv-import`. Routing a handler at a
wrong opcode is a silent failure: the packet is dispatched to the wrong
handler or dropped. Per-version IDA verification of both opcodes is a hard
prerequisite, not a nicety (FR-6.1).

## 5. Functional Requirements

### FR-1 — Serverbound codec and handler

- **FR-1.1** A new serverbound packet model for `ARAN_COMBO_COUNTER` in
  `libs/atlas-packet`, with both `Decode` and `Encode`, following the
  immutable-struct convention.
- **FR-1.2** The body is **empty** on v84 (verified). The design phase MUST
  confirm emptiness per version from each IDB before writing the codec; if any
  version carries a body, that version's decode is version-gated with the
  `MajorAtLeast` idiom, never a raw `> N` comparison.
- **FR-1.3** A new handler in
  `services/atlas-channel/atlas.com/channel/socket/handler/`, registered in
  all six seed templates under `socket.handlers` at its sorted `opCode`
  position (`tools/template-opcode-order-guard.sh`), with a non-empty
  validator (a handler with a missing validator is silently dropped — see the
  project's known-bug list).
- **FR-1.4** The handler is a pure request: it carries no client-supplied
  data, so it performs no decoding beyond the opcode.

### FR-2 — Server-side eligibility gates

The handler MUST re-derive every gate the client applied, from authoritative
server state, and MUST NOT trust that the client applied them. A request that
fails any gate is a silent no-op (debug-logged, no error to the client, no
`SHOW_COMBO`).

- **FR-2.1** The character's job is Aran (`AranStage1Id` 2100 … `AranStage4Id`
  2112) or Legend (`LegendId` 2000).
- **FR-2.2** The character has learned Combo Ability at level > 0 —
  `AranStage1ComboAbilityId` (`21000000`) for Aran jobs, or the Legend variant
  (`20000017`) for job 2000. See FR-5.1 for the constant.
- **FR-2.3** The character's equipped weapon is a polearm. `libs/atlas-constants`
  already exposes weapon-type resolution (`item.GetWeaponType`); the design
  phase resolves which enum member corresponds to the client's weapon type 44
  and verifies it against WZ item data rather than asserting it.
- **FR-2.4** Gate evaluation MUST NOT add a synchronous REST round trip per
  packet on the hot path beyond what is unavoidable; see NFR-1.

### FR-3 — Combo state

Combo count is stored as the `ARAN_COMBO` stat value on a buff owned by the
Combo Ability source id, in `atlas-buffs` — the same representation task-142
used for Warrior combo orbs.

- **FR-3.1** On the first eligible increment for a character with no active
  Combo Ability buff, the buff is **seeded** via `ApplyNoExpiry`
  (`services/atlas-channel/.../character/buff/processor.go:81`) carrying an
  `ARAN_COMBO` statup. Seeding is required because `UpdateStatValue`'s
  `INCREMENT` is a no-op when the buff is absent, expired, or lacks the stat
  (`services/atlas-buffs/.../character/registry.go:373-386`) — an unseeded
  counter would never move.
- **FR-3.2** Subsequent increments use `UpdateStatValue` with
  `StatOperationIncrement`, amount 1, and a cap (FR-3.3). The registry clamps
  to cap and floors at 1.
- **FR-3.3** The combo cap is derived from skill-effect data (Combo Ability's
  level-scaled effect), not hardcoded. The design phase MUST establish the cap
  source from WZ data via `atlas-data`; if no effect field governs it, the
  fallback and its justification are recorded in `design.md`. **Unverified at
  spec time.**
- **FR-3.4** Combo state is per-character and tenant-scoped. It does not
  survive a character logging out; the existing buff-lifecycle rules govern.

### FR-4 — Decay

- **FR-4.1** A periodic tick task in `atlas-buffs` decays combo for characters
  who have not landed a qualifying hit within the idle window. It follows the
  established shape: a `tasks.Register(l, ctx)(tasks.NewXTick(l, interval))`
  registration in `services/atlas-buffs/atlas.com/buffs/main.go:75`, a
  `TenantRegistry[uint32, time.Time]` of last-hit timestamps alongside
  `poisonTicks` (`character/registry.go:25,36`), and a
  `ProcessComboTicks`-style fan-out mirroring `ProcessPoisonTicks`
  (`character/processor.go:28`).
- **FR-4.2** Each accepted increment refreshes the character's last-combo
  timestamp.
- **FR-4.3** The idle window and the decay shape (full reset to zero vs. a
  step-down) are set in design. Cosmic's `AranComboHandler` resets after 3 s of
  idle; whether retail decays or hard-resets is **unverified** and MUST be
  established from client/WZ evidence before the value is fixed (FR-6.3).
- **FR-4.4** When combo decays to zero the Combo Ability buff is cancelled and
  the client is told — a stale on-screen counter is a visible bug.
- **FR-4.5** The tick task must not scan tenants with no Aran combo state.
  The `berserk` registry comment
  (`services/atlas-buffs/.../berserk/registry.go:23`) records this exact
  concern for its own ticker; combo follows the same discipline.

### FR-5 — Constants and static data

- **FR-5.1** Add the Legend Combo Ability constant (`20000017`) to
  `libs/atlas-constants/skill/constants.go` plus the per-version generated
  identity tables, following the existing Aran entries
  (`AranStage1ComboAbilityId` at `constants.go:3391`). Note: the task-187
  availability audit records job 2000's v79/v83 WZ skill set as
  `20001000`–`20001004` only, with **no `20000017`** — the id comes from the
  *client's* `GetSkillLevel` check (task-142 IDA read), not from a WZ dump.
  The design phase must reconcile this before deciding whether the Legend
  branch is reachable on any provisioned version, and record the outcome.
- **FR-5.2** `services/atlas-data/atlas.com/data/skill/reader.go:470` currently
  emits `produceBuffStatAmount(statups, character.TemporaryStatTypeAranCombo,
  100)` — a hardcoded 100 for `AranStage1ComboAbilityId`. What the client does
  with the `ARAN_COMBO` stat value is **unverified**. The design phase MUST
  determine, from `CUserLocal::OnIncComboResponse` and the temporary-stat
  encode path, whether the value is the live combo count, a tier index, or an
  opaque marker, and then either (a) make the reader emit the correct
  semantic value, or (b) document with evidence why 100 is correct and leave
  it. A change here without that evidence is not acceptable.
- **FR-5.3** No new temporary-stat type is needed:
  `character.TemporaryStatTypeAranCombo` exists
  (`libs/atlas-constants/character/temporary_stat.go:74`) and its wire bit is
  already allocated in the packet model
  (`libs/atlas-packet/model/character_temporary_stat.go:162`,
  `newAndIncNonDiseased` with no-op foreign writer/reader).

### FR-6 — Clientbound `SHOW_COMBO` and verification

- **FR-6.1** Before any codec is written, both opcodes are IDA-verified per
  version against that version's IDB (resolve the session from `idb_list` by
  binary **name**). Any correction is written back to
  `docs/packets/registry/<version>.yaml` with provenance, and the matrix
  regenerated.
- **FR-6.2** A clientbound `SHOW_COMBO` writer in `libs/atlas-packet` with
  `Encode` and `Decode`, registered in all six templates under
  `socket.writers` with an `fname` (seed writers require it) at its sorted
  `opCode` position. task-142's IDA read records the body as a 4-byte counter
  fed to `DrawCombo`; this MUST be re-derived per version rather than assumed.
- **FR-6.3** `SHOW_COMBO` is emitted to the acting character only (it drives
  their local HUD) on every accepted increment and on decay-to-zero.
- **FR-6.4** Each of the twelve matrix cells (2 ops × 6 versions) is promoted
  through the single-cell verify procedure
  (`docs/packets/audits/VERIFYING_A_PACKET.md`, via `/verify-packet` /
  `packet-verifier`), with a byte fixture and a pinned evidence record. A cell
  that does not promote is a failure, not a prose claim.

## 6. API Surface

No new REST endpoints. The feature uses existing surfaces:

**Kafka — `COMMAND_TOPIC_CHARACTER_BUFF` (existing):**

- `APPLY_NO_EXPIRY` — seeds the Combo Ability buff with an `ARAN_COMBO` statup
  (FR-3.1). Emitted via
  `buff.Processor.ApplyNoExpiry(f, fromId, sourceId, level, statups)`.
- `UPDATE_STAT_VALUE` — increments the count (FR-3.2). Body shape is pinned by
  a canonical-JSON test on both sides
  (`services/atlas-buffs/.../kafka/message/character/kafka_test.go:66`,
  `services/atlas-channel/.../kafka/message/buff/kafka_test.go:26`):
  `{"sourceId":…,"statType":"ARAN_COMBO","operation":"INCREMENT","amount":1,"cap":N}`.
- `CANCEL` — clears the buff on decay-to-zero (FR-4.4).

Any new command body added for decay must respect the duration-unit contract
enforced by `tools/buff-duration-guard.sh` (milliseconds).

**Socket — two packet types**, one each direction, described in §5.

## 7. Data Model

No relational schema changes.

**New in-memory / Redis state (atlas-buffs):** a tenant-scoped registry of
last-combo-hit timestamps, `TenantRegistry[uint32, time.Time]`, keyed by
character id — structurally identical to `poisonTicks`
(`services/atlas-buffs/.../character/registry.go:25,36`), including the
key-prefix convention (`"buffs-poison"` → an analogous combo prefix). All
keyed Redis access goes through `libs/atlas-redis`
(`tools/redis-key-guard.sh`).

**Combo count itself is not new state** — it is the `ARAN_COMBO` stat amount on
the existing per-character buff registry entry, reached by `srcKey(sourceId)`.

**Tenant configuration:** six socket-config seed templates gain one handler
entry and one writer entry each. Live tenants must be reconciled to the new
templates, or the opcodes will be absent from the running config and the
packets silently dropped — a repeatedly-observed failure mode in this project.

## 8. Service Impact

| Service / lib | Change |
|---|---|
| `libs/atlas-packet` | New `ARAN_COMBO_COUNTER` serverbound model and `SHOW_COMBO` clientbound model, each with `Encode` + `Decode` and version gates via `MajorAtLeast`. |
| `libs/atlas-constants` | Add Legend Combo Ability `20000017` to `skill/constants.go` and the per-version identity tables (FR-5.1). |
| `atlas-channel` | New handler for the serverbound op; eligibility gates (job / learned skill / weapon); emits the buff commands and writes `SHOW_COMBO`. |
| `atlas-buffs` | Combo decay tick task, last-hit timestamp registry, decay processing and cancel-on-zero. `UpdateStatValue` itself needs no change. |
| `atlas-data` | Resolve the `ARAN_COMBO` statup value semantics at `skill/reader.go:470` (FR-5.2); possibly expose the combo cap from Combo Ability's effect (FR-3.3). |
| `atlas-configurations` | Handler + writer entries in six seed templates. |
| `docs/packets` | Per-version opcode verification, registry corrections, twelve verified matrix cells, evidence records. |

Unaffected: `atlas-character`, `atlas-character-factory`, `atlas-monster`,
`atlas-ui`.

## 9. Non-Functional Requirements

- **NFR-1 — Hot path cost.** `RequestIncCombo` fires from `CMob::OnHit`, so an
  Aran in a mob-dense map generates it at melee-hit frequency. The handler MUST
  NOT perform an unbounded number of synchronous REST reads per packet. The
  design phase states the exact per-packet cost (which lookups are cached in
  the session/character model vs. fetched) and justifies it — task-166's PRD
  (§1.1) shows this project treats per-attack fetch counts as a first-class
  design concern. If the naive gate evaluation is too costly, the design must
  propose a cheaper ordering (cheapest, most-selective gate first) or caching.
- **NFR-2 — Client spam tolerance.** A client that sends the op while
  ineligible, or faster than the mechanic allows, is rejected without error
  logging at warn/error level and without emitting Kafka commands. Ineligible
  rejection must be decidable from data already in hand wherever possible.
- **NFR-3 — Multi-tenancy.** All state is tenant-scoped via
  `tenant.MustFromContext(ctx)`. The decay ticker fans out per tenant and skips
  tenants with no combo state (FR-4.5).
- **NFR-4 — Failure isolation.** Combo bookkeeping never fails a player action.
  All emit failures are logged and swallowed, matching `comboOrbTryUpdate`'s
  contract (`character_attack_combo.go:166`, whose doc comment states "All
  failures are logged and swallowed — the attack pipeline never fails on orb
  bookkeeping").
- **NFR-5 — Observability.** Debug-level logging on accepted increments, gate
  rejections (with the failing gate), and decay events, carrying character id
  and resulting count.
- **NFR-6 — Version isolation.** No wire change to any already-verified
  version. Version divergence is expressed with the `MajorAtLeast` idiom, never
  raw major-version comparisons (the `MajorVersion() > 83` off-by-one is a
  known project bug).
- **NFR-7 — Goroutines.** Any concurrency goes through `routine.Go`
  (`tools/goroutine-guard.sh`).

## 10. Open Questions

These are the items this PRD deliberately leaves unresolved. Each is a
**verification** task for the design phase, not a judgment call — none may be
closed by assertion.

1. **Serverbound opcodes for v83/v87/v92/v95/jms185.** All `csv-import`. v84's
   verified value moved from the csv value. IDA-verify each. (FR-6.1)
2. **Clientbound `SHOW_COMBO` opcodes and body.** All `csv-import`; the 4-byte
   body is a task-142 IDA read on one version. Re-derive per version. (FR-6.2)
3. **Is the serverbound body truly empty on every in-scope version?** Verified
   only for v84. (FR-1.2)
4. **What is the `ARAN_COMBO` stat value?** `reader.go:470` hardcodes 100.
   Determine from `OnIncComboResponse` and the temporary-stat encode path
   whether it is the live count, a tier, or opaque. (FR-5.2)
5. **Combo cap.** Which WZ/effect field governs it, if any. (FR-3.3)
6. **Decay window and shape.** Cosmic uses a 3 s reset; retail behavior
   unverified. Determine whether combo hard-resets or steps down. (FR-4.3)
7. **Combo Ability milestones.** Cosmic re-applies the Combo Ability effect at
   every 10 combo (`StatEffect.applyComboBuff`). Whether Atlas needs discrete
   milestone re-application, or whether a single seeded buff whose stat value
   tracks the count is sufficient, depends on answer 4.
8. **Is `20000017` reachable?** The task-187 availability audit lists job 2000's
   v79/v83 WZ skills as `20001000`–`20001004` with no `20000017`. If the id is
   client-only and never learnable on a provisioned version, the Legend branch
   of FR-2.2 is dead code and should be omitted with that finding recorded.
   (FR-5.1)
9. **Polearm weapon-type mapping.** Confirm which `libs/atlas-constants`
   weapon-type member corresponds to the client's type 44, against WZ item
   data. (FR-2.3)

## 11. Acceptance Criteria

**Wire and routing**

- [ ] Both opcodes IDA-verified for all six versions; any registry correction
      committed with provenance and the matrix regenerated.
- [ ] `ARAN_COMBO_COUNTER` serverbound model exists in `libs/atlas-packet` with
      `Encode` and `Decode`, version-gated with `MajorAtLeast` where it
      diverges.
- [ ] `SHOW_COMBO` clientbound model exists with `Encode` and `Decode`.
- [ ] Handler entry (with a non-empty validator) and writer entry (with
      `fname`) added to all six seed templates at sorted `opCode` positions.
- [ ] All twelve matrix cells (2 ops × 6 versions) show `✅` in
      `docs/packets/audits/STATUS.md`, each backed by a byte fixture carrying a
      `packet-audit:verify` marker and a pinned evidence record.

**Behavior**

- [ ] An Aran with Combo Ability and a polearm who hits a monster sees the
      combo counter increment on screen.
- [ ] The first eligible increment seeds the Combo Ability buff; subsequent
      ones increment its `ARAN_COMBO` stat value, clamped at the cap.
- [ ] Combo does not exceed the cap however fast the client sends the op.
- [ ] After the idle window with no qualifying hits, combo decays; on reaching
      zero the buff is cancelled and the client's counter is cleared.
- [ ] A qualifying hit refreshes the idle timer.
- [ ] Each of these is a silent no-op with no Kafka emission: non-Aran/Legend
      job; Aran without Combo Ability learned; Aran with a non-polearm weapon.
- [ ] `ARAN_COMBO` statup semantics at `reader.go:470` are either corrected or
      justified in writing with cited evidence.
- [ ] `20000017` added to `libs/atlas-constants` (or its omission recorded with
      the FR-5.1 finding).

**Verification (per CLAUDE.md §Build & Verification)**

- [ ] `go test -race ./...` clean in every changed module.
- [ ] `go vet ./...` clean in every changed module.
- [ ] `go build ./...` clean in every changed service.
- [ ] `docker buildx bake atlas-<svc>` clean for every service whose `go.mod`
      was touched.
- [ ] `tools/redis-key-guard.sh` clean (new combo timestamp registry).
- [ ] `tools/goroutine-guard.sh` clean.
- [ ] `tools/lint.sh --check` clean.
- [ ] `tools/template-opcode-order-guard.sh` clean (six templates changed).
- [ ] `tools/template-duplicate-binding-guard.sh` clean.
- [ ] `tools/buff-duration-guard.sh` clean.
- [ ] `tools/skill-job-id-guard.sh` clean.
- [ ] `docs/TODO.md` updated to reflect the landed state. The missing-features
      research corpus (`docs/research/missing-features/skills-and-buffs.md` §7,
      `new-jobs-and-version-delta.md` §5) is untracked in the main repo — if it
      is committed by then, update both entries there too, including the
      v84→v83 version correction (§1.2).
- [ ] Code review run (`superpowers:requesting-code-review`) before the PR.
