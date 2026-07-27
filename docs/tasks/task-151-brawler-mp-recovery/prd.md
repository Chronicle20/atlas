# MP Recovery (Brawler 5101005) — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-07-10
---

## 1. Overview

MP Recovery (skill id 5101005) is a 2nd-job Brawler active skill that sacrifices a fixed
fraction of the caster's max HP and restores MP proportional to the HP lost. In Atlas today
the cast animation plays and the cooldown applies, but nothing happens to the caster's HP or
MP — the skill is functionally dead. This task implements the server-side effect.

The client is entirely passive for this skill (IDA-verified against GMS v83,
`MapleStory_dump.exe`): the only 5101005-specific client code is a routing branch in
`CUserLocal::DoActiveSkill` (0x96821B) that sends the cast down the generic stat-change path
after a `SKILLENTRY::IsCorrectWeaponType` check. The generic consume gate
(`CSkillInfo::CheckConsumeForActiveSkill`, 0x764256) only rejects a cast when
`hpCon >= currentHP`, and 5101005 has no `hpCon` node in WZ — so the client permits the cast
at any HP, applies no HP/MP change locally, and simply renders whatever stat changes the
server pushes. The entire effect is server-authoritative.

Reference behavior is Cosmic `SpecialMoveHandler.java:118-124`: on cast, lose
`MaxHP / x` HP and gain `lose * y / 100` MP, where `x` and `y` come from the skill's
per-level WZ data. Per owner decision, Atlas mirrors Cosmic exactly, including the unguarded
HP loss — a caster at or below `MaxHP / x` current HP dies from the cast.

## 2. Goals

Primary goals:
- Casting MP Recovery reduces the caster's HP by `MaxHP / x` and increases MP by
  `(MaxHP / x) * y / 100`, driven by the tenant version's skill data (never hardcoded values).
- Behavior matches Cosmic parity, including the low-HP edge (HP can reach 0 → death via the
  existing HP-change path).
- Follow the established per-skill handler pattern (registry + `init()` registration), as
  Heal (task-045) and Mystic Door (task-093) do.

Non-goals:
- Chakra (4211001) — item 5 of the same audit; separate task.
- Other pirate skill gaps (Energy Charge, etc.).
- Any packet, opcode, or client-visible protocol change — none is needed; the existing
  stat-changed flow from atlas-character already renders the result.
- Guarding or rejecting low-HP casts (explicitly decided against; Cosmic parity).

## 3. User Stories

- As a Brawler player, I want casting MP Recovery to convert a slice of my max HP into MP so
  that I can sustain my MP mid-fight the way the skill is described.
- As a Brawler player at critically low HP, I expect the cast to behave exactly like classic
  servers (the HP cost applies unguarded), so the skill is not silently safer than authentic
  behavior.
- As a server operator, I want the formula driven by each tenant version's skill data so a
  future version with different `x`/`y` values needs no code change.

## 4. Functional Requirements

FR-1 — Per-skill handler registration
- A new handler subpackage `skill/handler/mprecovery` in atlas-channel registers itself for
  `skill.BrawlerMPRecoveryId` (5101005, already defined at
  `libs/atlas-constants/skill/constants.go:3194`) via `init()` in the existing registry
  (`skill/handler/registry.go`), with a blank import added to
  `skill/handler/registrations/registrations.go`.
- The handler runs from the existing per-skill dispatch point in
  `skill/handler/common.go` (`Lookup(...)` at the end of `UseSkill`); the generic path
  (cooldown from WZ `cooltime`, no statups since the skill has no `duration`) is unchanged.

FR-2 — Effect formula (Cosmic parity)
- `hpLost = floor(MaxHP / x)`, `mpGain = floor(hpLost * y / 100)`, integer arithmetic,
  where `MaxHP` is the caster's max HP and `x`/`y` are the cast level's effect attributes.
- WZ v83 ground truth (`Skill.wz/510.img.xml`, verified): max level 10; `x = 10` at every
  level; `y` scales 55 (L1) → 75 (L5) → 100 (L10); `cooltime` 70s → 50s → 25s. The
  implementation MUST read `x`/`y` from the effect model, not these constants.
- Apply `ChangeHP(field, casterId, -hpLost)` then `ChangeMP(field, casterId, +mpGain)` via
  the existing `character.Processor` (same calls the dispatcher and Heal already use).
- `mpGain` is computed from the full `hpLost` amount (Cosmic computes gain from the intended
  loss, not the post-clamp delta).
- Magnitudes fit the existing `int16` delta parameters: max HP is bounded well below
  `32767 * x`, so `MaxHP / x` cannot overflow.

FR-3 — Low-HP edge (owner-decided)
- No guard. If `currentHP <= hpLost` the existing ChangeHP semantics apply (HP floors at 0,
  character dies through the same path as damage). No special-case code in the handler.

FR-4 — Effect model exposure
- `data/skill/effect/model.go` in atlas-channel gains a public `Y()` getter. The `y` field
  already exists in the struct and is already populated from REST
  (`data/skill/effect/rest.go:115`); only the getter is missing. `X()` already exists.
- atlas-data requires no change: its reader already ingests `x` and `y` generically
  (`services/atlas-data/atlas.com/data/skill/reader.go:208-209`).

FR-5 — Failure handling
- If loading the caster fails, log and return an error without emitting HP/MP changes
  (mirroring Heal's error handling); never apply a partial effect where MP is gained without
  the HP cost having been requested.

## 5. API Surface

None. No new or modified REST endpoints, Kafka topics, or packet writers/handlers. The
effect rides entirely on the existing `character.Processor.ChangeHP` / `ChangeMP` command
emission and the existing stat-changed client packet flow.

## 6. Data Model

None. No new entities, fields, or migrations. Skill effect data (`x`, `y`, `cooltime`) is
already ingested per tenant version by atlas-data and served to atlas-channel.

## 7. Service Impact

- **atlas-channel** (only service touched):
  - New `skill/handler/mprecovery/` subpackage (handler + tests).
  - One-line blank import in `skill/handler/registrations/registrations.go`.
  - `Y()` getter added to `data/skill/effect/model.go`.
- **atlas-data**: no change (x/y already ingested and served).
- **libs/atlas-constants**: no change (`BrawlerMPRecoveryId` already exists).

## 8. Non-Functional Requirements

- Multi-tenancy: handler operates through the tenant-scoped `context.Context` exactly like
  existing handlers; formula inputs come from the tenant version's skill data, so v83–v95
  tenants each get their own `y` progression with zero version-conditional code.
- Observability: log the cast outcome (caster id, level, hpLost, mpGain) at debug level,
  errors at error level, consistent with Heal.
- Testing: use the project Builder pattern for test setup (no `*_testhelpers.go`); formula
  tests plus handler tests using the existing seam-override style in `skill/handler`
  (`loadCasterFunc` precedent).

## 9. Open Questions

None. The two open decisions from the interview were resolved:
- Task number 151 (backlog pre-assignment honored over `task-numbers.sh next` = 150, which
  is exposed to the live concurrent-numbering race in this batch — see the duplicate
  task-149 worktrees).
- Low-HP edge: Cosmic parity, unguarded (owner-selected after IDA verification that the
  client has no gate).

## 10. Acceptance Criteria

- [ ] Casting 5101005 at level L applies `-floor(MaxHP/x)` HP and `+floor(floor(MaxHP/x)*y/100)`
      MP, with `x`/`y` read from the tenant's skill effect data for level L.
- [ ] Cast at `currentHP <= MaxHP/x` still applies the full HP loss (HP reaches 0 → existing
      death handling) and the full MP gain.
- [ ] Handler is registered via `init()` + blank import; casting any other skill is
      unaffected (registry lookup miss → no-op, as today).
- [ ] `effect.Model.Y()` exposed and covered by the handler tests.
- [ ] Unit tests: formula (levels 1/5/10 against WZ-verified v83 values), error path
      (caster load failure → no emits), and registration.
- [ ] `go test -race ./...`, `go vet ./...`, `go build ./...` clean in atlas-channel;
      `tools/redis-key-guard.sh` clean from repo root.
- [ ] Code review (`superpowers:requesting-code-review`) run before PR.
