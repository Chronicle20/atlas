# Plan — Hex of the Beholder authentic buff accumulation

Status: Ready for implementation (task-088 addendum)
Design: `addendum-hex-accumulation-design.md`
Modules touched: `services/atlas-buffs`, `services/atlas-summons`. **No `atlas-channel`,
no `libs/` change.**

Build/verify gate for both modules (per CLAUDE.md): `go test -race ./...`, `go vet ./...`,
`go build ./...`, `docker buildx bake atlas-buffs` **and** `docker buildx bake atlas-summons`
from the worktree root, and `tools/redis-key-guard.sh` clean.

---

## Phase 0 — confirm contract & callers (no code)

0.1 `grep -rn '\.Buffs()' services/atlas-buffs` — enumerate callers of the character
`Model.Buffs()` getter (REST handler, projection, tests) that the map-key-type change in
Phase 2 must update.
0.2 Confirm the two `ApplyCommandBody` declarations are the only producers of `APPLY`:
`services/atlas-buffs/.../kafka/message/character/kafka.go` (consumer) and
`services/atlas-summons/.../buff/producer.go` (the only summons producer). Grep other
services for `COMMAND_TOPIC_CHARACTER_BUFF` producers; if any other producer exists it does
**not** set `accumulate` (defaults false) and is unaffected — note it, don't change it.

## Phase 1 — contract: add `accumulate` flag

1.1 `atlas-buffs` `kafka/message/character/kafka.go`: add
`Accumulate bool \`json:"accumulate,omitempty"\`` to `ApplyCommandBody`.
1.2 `atlas-buffs` `kafka/consumer/character/consumer.go:handleApply`: pass
`c.Body.Accumulate` through to `Processor.Apply(...)` (new trailing param).
1.3 `atlas-summons` `buff/producer.go`: add the identical `Accumulate` field to its mirrored
`ApplyCommandBody`; thread an `accumulate bool` param through `applyProvider`/`ApplyProvider`;
update the "MUST stay byte-identical" comment.
- **Verify:** both modules `go build`. Add a JSON round-trip test (a stat-level test in each
  module) asserting `accumulate` omitted ⇒ absent in JSON, and `true` ⇒ present, identical
  across mirrors.

## Phase 2 — atlas-buffs: per-stat accumulate storage

2.1 `character/model.go`: change `buffs map[int32]buff.Model` → `map[string]buff.Model`;
update `Buffs()` return type, `MarshalJSON`/`UnmarshalJSON` structs. Update Phase-0 callers
of `Buffs()` to the string key (values unchanged).
2.2 `character/registry.go`: add key helpers
`srcKey(sourceId int32) string` and `statKey(sourceId int32, statType string) string`.
Replace `make(map[int32]…)`/`m.buffs[sourceId]` accordingly. Existing value-iterating methods
(`Cancel`, `GetExpired`, `CancelAll`, `CancelByStatTypes`, `GetPoisonCharacters`,
`HasImmunity`) keep their logic — only the key var type changes.
2.3 `character/registry.go:Apply` — add `accumulate bool` param:
- `false`: `m.buffs[srcKey(sourceId)] = NewBuff(sourceId, level, duration, changes)` (today's
  behavior). Return the single buff.
- `true`: for each `c` in `changes`, `m.buffs[statKey(sourceId, c.Type())] =
  NewBuff(sourceId, level, duration, []stat.Model{c})`. Return the list of per-stat buffs.
  (Signature: return `[]buff.Model` or keep `buff.Model` + a variant — see 2.4.)
2.4 `character/processor.go:Apply` — add `accumulate bool` param; in accumulate mode emit one
`appliedStatusEventProvider(...)` **per stat buff** (each with its own `expiresAt`); default
mode unchanged (one APPLIED). Keep the disease-immunity short-circuit.
2.5 Update the `Processor` interface signature and any mocks.
- **Tests** (`character/*_test.go`), all new + regression:
  - accumulate: Apply pdd then mdd under 1320009 ⇒ 2 entries; both in `Get`; independent
    `ExpiresAt`. Re-apply pdd ⇒ pdd timer refreshed, mdd untouched.
  - `GetExpired` with pdd expired, mdd live ⇒ returns only pdd; mdd remains.
  - `Cancel(1320009)` ⇒ removes both; `CancelByStatTypes({WEAPON_DEFENSE})` ⇒ removes pdd only.
  - **regression:** default Apply of [A,B,C] then recast [A,B,C] ⇒ 1 entry, overwritten (not
    accumulated); recast carrying the same set refreshes. Existing buff tests green.
- **Verify:** `go test -race ./...`, `go vet`, redis-key-guard, `docker buildx bake atlas-buffs`.

## Phase 3 — atlas-summons: one random stat per pulse

3.1 `summon/beholder_task.go`: add an injectable `pick func(n int) int` field to
`BeholderTask` (default `rand.Intn`), mirroring the `emit` field pattern.
3.2 `sweepBuff`: replace the all-changes emit with:
- `pool := m.BuffChanges()`; if empty, skip (unchanged guard).
- `c := pool[t.pick(len(pool))]`.
- emit `buffmsg.ApplyProvider(m.Field(), owner, owner, m.BuffSourceId(), m.BuffLevel(),
  m.BuffDuration(), []buffmsg.StatChange{{Type:c.Type, Amount:c.Amount}}, /*accumulate=*/true)`.
- keep the `skillEventProvider` animation pulse + timer advance unchanged.
3.3 `buff/producer.go`: already updated in 1.3 — pass `accumulate:true`.
- **Tests** (`beholder_task_test.go`): update `TestBeholderSweepFiresHealAndBuffWhenDue` —
  inject `pick` returning a fixed index; assert exactly one `APPLY` with one change and
  `Accumulate:true`; assert the heal + SKILL-pulse assertions still hold. Add a test iterating
  `pick` across the pool asserting union coverage and that the SourceId stays positive 1320009.
- **Verify:** `go test -race ./...`, `go vet`, redis-key-guard, `docker buildx bake atlas-summons`.

## Phase 4 — integration & build

4.1 Full gate on both changed modules (test/vet/build/bake/redis-key-guard).
4.2 Code review (`superpowers:requesting-code-review` → backend-guidelines-reviewer +
plan-adherence-reviewer) before PR; write findings to `audit.md`.

## Phase 5 — live verification (ephemeral env)

5.1 Deploy; DrK with Hex ≥ L20. Summon Beholder. Observe buff icons appear **one at a time**
across pulses with independent countdowns (not a lockstep refresh).
5.2 Let one stat lapse → exactly one icon drops; others persist. Relog → per-stat restore
with remaining timers.
5.3 Loki cross-check: per-pulse `APPLIED{changes:[oneStat]}` and independent
`EXPIRED{changes:[oneStat]}`; confirm no `record not found` / no client disconnect.
5.4 Regression smoke: a normal potion/skill buff still applies and overwrites on recast
(no accidental accumulation).

---

## Risk register

- **Shared-service map-key change (Phase 2.1):** blast radius is the `Buffs()` callers; bound
  it via Phase 0.1 enumeration + regression tests. Value shapes (`buff.Model`, events, REST)
  are untouched.
- **Contract drift:** the two `ApplyCommandBody` mirrors must stay byte-identical (Phase 1
  round-trip test guards this).
- **Two-module bake:** both `atlas-buffs` and `atlas-summons` `go.mod` are effectively touched
  (summons via producer change) — bake both; CI catches a missed Dockerfile COPY otherwise.
- **Non-goal creep:** do not alter the heal path, hex values, or channel code.

## Out of scope / explicitly not done

- Any `atlas-channel` change (already per-stat).
- Global `(sourceId,statType)` re-key of all buffs (rejected in design §3 / addendum §5.2 —
  opt-in flag instead).
- Changing hex interval/duration/stat values.
