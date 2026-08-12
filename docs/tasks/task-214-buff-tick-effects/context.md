# task-214 — Implementation Context

Companion to [`plan.md`](plan.md). Everything an implementer needs that is not
in the plan's task bodies: where the code lives, why the design landed where it
did, and the traps.

Worktree: `.worktrees/task-214-buff-tick-effects` · branch `task-214-buff-tick-effects`
Artifacts: [`prd.md`](prd.md) · [`design.md`](design.md) · [`plan.md`](plan.md)

---

## 1. Blast radius

Exactly one service: **atlas-buffs**, under
`services/atlas-buffs/atlas.com/buffs/`. No other service, library, template,
deploy manifest, or packet codec changes.

atlas-data is deliberately untouched: design.md §2 verified against the local
v83 WZ dump that `skill/reader.go:342` (`DRAGON_BLOOD` ← `e.X()`) and
`reader.go:318` (`RECOVERY` ← `e.X()`) already store the per-tick magnitude the
tick path needs. PRD FR-3.3 and FR-4.3 made that verification a precondition; it
came back clean, so the "correct reader.go" branch of those FRs is a no-op.

---

## 2. The code as it stands today

### The poison path being replaced

| Concern | File:line | Notes |
|---|---|---|
| Scan | `character/registry.go:300-328` `GetPoisonCharacters` | Compares `c.Type() == "POISON"`, `break`s after the first match per buff, returns one entry per *buff*. |
| Throttle store | `character/registry.go:25, 36-38` `poisonTicks` | `TenantRegistry[uint32, time.Time]`, namespace `buffs-poison`, keyed by character alone. |
| Store accessors | `registry.go:330-347` | `GetLastPoisonTick`, `UpdatePoisonTick`, and `ClearPoisonTick` — **the last has zero callers**, which is exactly the bug FR-6.1 exists to prevent recurring. |
| Tick pass | `character/processor.go:253-279` `ProcessPoisonTicks` | Hard-coded `time.Second` throttle, `time.Now()` read inline (untestable), `amount >= 0 { continue }` guard. |
| Tenant fan-out | `processor.go:281-296` | `GetTenants` → `routine.Go` → `tenant.WithContext`. Copy this shape verbatim. |
| Ticker task | `tasks/poison.go` | Wired at `main.go:75` as `tasks.NewPoisonTick(l, 1000)`. |

There are **no existing tests of the poison tick's emit semantics**. `tasks/poison_test.go`
only covers `SleepTime` math and a no-panic `Run`. So PRD's "ported, not deleted"
for POISON parity means: the three task-level tests port to `periodic_test.go`,
and the actual poison behavior gets its *first* assertions in
`character/periodic_processor_test.go` (plan Task 3,
`TestPeriodicTickPoisonParity`). Do not go looking for a poison emit test to
port — there isn't one.

### The patterns to copy

- **Injected-dependency processor:** `berserk/ProcessorImpl`
  (`berserk/processor.go:36-79`) carries `now func() time.Time` plus stubbed
  external readers, and `berserk/processor_test.go:20-34` builds it as a
  same-package struct literal. `character.ProcessorImpl` gets the same treatment
  (`now`, `getCharacterHp`). This is the project's answer to "no
  `*_testhelpers.go`, no test-only constructors."
- **HP read:** `extchar.RequestById(characterId)(l, ctx)` →
  `extchar.RestModel{Level byte; Hp uint16}` (`external/character/rest.go:7-11`),
  already used by `berserk/processor.go:54`.
- **Emit:** `message.Emit(l, ctx)(func(buf *message.Buffer) error {...})` with
  `buf.Put(topic, provider)`. The provider already exists:
  `changeHPCommandProvider(worldId, channelId, characterId, amount int16)`
  (`character/producer.go:44-58`), keyed `producer.CreateKey(int(characterId))`.
- **Removal paths** all already build a `sets [][]stat.Model` right before
  calling `markBerserkDirtyOnMaxHpChange(...)` — that line is the hook point for
  `ClearPeriodicTicksFor`, so the four call sites need one added line each, not
  restructuring.

### Redis

`libs/atlas-redis.TenantRegistry[K, V]` is generic over the key with a
caller-supplied `keyFn func(K) string`, so a struct key (`TickKey`) works with
no new type: `atlas.NewTenantRegistry[TickKey, time.Time](client, "buffs-tick", ...)`.
`PutWithTTL` exists (`libs/atlas-redis/tenant_registry.go:117`) — the design's
5-minute TTL needs no new library surface. Using `TenantRegistry` (never the raw
`*goredis.Client`) is what keeps `tools/redis-key-guard.sh` green.

---

## 3. Decisions inherited from design.md (do not re-litigate)

| # | Decision | Why |
|---|---|---|
| D1 | Table in a new pure `periodic` package; scan + emit stay in `character` | The buff store lives in `character.Registry`; a separate processor would duplicate the scan or force `character` to export its buff map. The diff stays close to a rename. |
| D2 | One composite-keyed registry, not one per stat type | One registry per row is the per-effect ceremony this task deletes. |
| D3 | One 1 s driving task with per-row interval gates | Preserves POISON's cadence exactly, keeps one scan pass. A 4 s row driven at 1 s drifts <1 s — same property poison has today; HP-per-4-s is not a tight-tolerance contract. |
| D4 | Floor by reducing the amount, not by cancelling the buff | PRD FR-3.4/3.5. Also avoids atlas-buffs inventing a cancel reason and firing an `EXPIRED` the client never asked for. |
| D5 | Fail closed on the HP read | One missed 4 s tick is invisible; one unintended `DIED` is not. |
| D6 | Recovery emits unconditionally — no overheal suppression | Suppressing costs an `exteffstats` call per healing character per 5 s. WZ says Recovery is `time`=30 s on `cooltime`=120 s ⇒ at most 6 ticks per cast, not a heartbeat. PRD FR-4.5's explicit fallback. |
| D7 | Dedupe duplicate `(character, statType)` by max amount | First-wins over a Go map is nondeterministic. |
| D8 | TTL on tick keys *in addition to* explicit clears | FR-6.1 requires the wiring; the TTL bounds a future removal path that forgets it. |

**Dragon Blood's WZ `desc`** ("decreases HP steadily until 4 seconds before the
remaining HP exhausts") implies cancel-on-exhaust, not floor-at-1. The PRD chose
floor-at-1 deliberately. Recorded in design.md §2 Q1 so nobody re-derives it from
the string and files it as a bug.

---

## 4. Traps

1. **Buff duration is milliseconds, and must be > 0.**
   `buff.NewBuff` (`buff/model.go:144-159`) does
   `expiresAt = now.Add(time.Duration(duration) * time.Millisecond)` and returns
   `ErrInvalidDuration` for `duration <= 0`. A fixture written as "60" is a
   60-millisecond buff that lapses mid-test; "-1" is an error, not an expired
   buff. Live fixtures use `600000`; a lapsed fixture is duration `1` plus a
   `10 * time.Millisecond` sleep.
2. **`buff.Model.Expired()` reads the real wall clock**, not the processor's
   injected `now`. Freezing the clock does not age a buff out. This is why the
   expiry fixtures sleep.
3. **`character/testmain_test.go` currently installs the *no-op* producer.**
   Asserting the emitted `CHANGE_HP` amount requires `InstallCapturing`. It is
   installed once per package from `TestMain` (`producer.Manager` caches one
   writer per topic for the singleton's lifetime); each test calls
   `emitted.Reset()` at the top, never re-installs. Consequence: the `character`
   package's tests must not run `t.Parallel()` — the capture is process-wide.
4. **The topic key in the capture is the env-var token itself.** Topics resolve
   through `topic.EnvProvider`, which falls back to the token name when the
   variable is unset, so tests read
   `emitted.Messages(character2.EnvCommandTopicCharacter)` — literally
   `"COMMAND_TOPIC_CHARACTER"`.
5. **`int16` conversion.** `ChangeHPCommandBody.Amount` is `int16` while
   `stat.Model.Amount()` is `int32`. Today's poison code converts without a
   bound; the plan adds a package constant `maxTickMagnitude = int32(32767)`
   (`math.MaxInt16`, written as a literal so `processor.go` needs no new import)
   so a corrupt stored amount
   degrades to a large tick instead of wrapping sign and turning a drain into a
   heal. No real WZ value comes near it.
6. **The non-positive-magnitude guard is a generalization, not a new rule.**
   Today's `amount := int16(-entry.Amount); if amount >= 0 { continue }` skips
   both zero *and negative* stored amounts. `if magnitude <= 0 { continue }`
   reproduces that for every row, including `Restore` rows where a negative
   stored amount would otherwise drain.
7. **Delete `ProcessPoisonTicks` from the `Processor` *interface*, not just the
   impl.** `var _ Processor = (*ProcessorImpl)(nil)` (`processor.go:43`) will
   catch the impl, but a stale interface method compiles fine and leaves dead
   surface.
8. **`berserk/processor.go:268`** carries a prose reference to
   `character.ProcessPoisonTicks`. A comment, so nothing breaks — but it goes
   stale in the same commit that removes the function.

---

## 5. Accepted, documented races

Both need an atlas-character contract change that PRD §7 places out of scope
("atlas-character: None"). They are recorded so a reviewer does not read them as
oversights.

- **Throttle vs. emit (design.md §3.5).** `GetPeriodicTick` and
  `UpdatePeriodicTick` straddle `buf.Put`, exactly as the poison path does. A
  crash between them re-ticks next pass; a crash before `Put` skips a tick. Both
  are one-interval errors on a non-idempotent HP mutation. Exactly-once needs an
  idempotency key on `CHANGE_HP`.
- **Dragon Blood HP snapshot (design.md §3.6).** The floor is computed from a
  read that is one Kafka hop old. Concurrent damage can still land the sum on 0
  and emit `DIED`. Closing it needs a "cannot be lethal" flag on
  `ChangeHPCommandBody`. Only reachable for a character already taking lethal
  damage that instant.

---

## 6. FR → task map

| Requirement | Where |
|---|---|
| FR-1.1–1.3 (table, single naming site, compile-time intervals) | Task 1 |
| FR-1.4 (magnitude is the stored `Amount()`, no atlas-data call) | Task 2 scan + Task 3 emit |
| FR-1.5 (zero/non-positive magnitude skipped) | Task 3 |
| FR-2.1 (generic scan replaces `GetPoisonCharacters`) | Task 2, Task 5 (deletion) |
| FR-2.2 (`(characterId, statType)` store) | Task 2 |
| FR-2.3 (one task, per-row gates) | Task 3, Task 5 |
| FR-2.4 (POISON parity) | Task 3 `TestPeriodicTickPoisonParity`, `TestPeriodicTickCommandShape` |
| FR-2.5 (multi-tenant, `routine.Go`) | Task 3 `ProcessPeriodicTicks` fan-out |
| FR-2.6 (`message.Emit`, keyed by characterId) | Task 3 (existing provider) |
| FR-3.1–3.3 (Dragon Blood row, 4 s, WZ magnitude) | Task 1 |
| FR-3.4–3.5 (floor at 1, buff keeps ticking) | Task 3 `TestPeriodicTickDragonBloodFloorsAtOne` |
| FR-3.6 (HP read via `extchar`, once per character per pass) | Task 3 `hpFor` + `TestPeriodicTickHpReadIsMemoizedPerPass` |
| FR-4.1–4.4 (Recovery row, 5 s, positive unclamped) | Task 1, Task 3 |
| FR-4.5 (overheal decision) | Recorded as design D6; no code |
| FR-5.1–5.5 (audit sweep) | Already committed: design.md §5 |
| FR-6.1 (clear on cancel/expire) | Task 4 |
| FR-6.2 (no residual keys) | Task 4 + the `PeriodicTickTTL` backstop |
| FR-6.3 (channel/world change resets nothing) | Structural — keys are character-scoped, world/channel are read fresh from the character model on each scan |
| PRD §10 verification checklist | Task 6 |

---

## 7. Verification

From `services/atlas-buffs/atlas.com/buffs`: `go build ./...`, `go vet ./...`,
`go test -race ./...`.

From the worktree root: `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`,
`tools/lint.sh --check` (run `tools/lint.sh` with no flags to fix formatting in
place first; it needs nvm on PATH or it false-fails on the frontend leg).

`go.mod` is not expected to change — no new dependency. If it does,
`docker buildx bake atlas-buffs` from the worktree root becomes mandatory.

Not applicable, no run needed: the template guards (opcode order, duplicate
binding, movement types), `skill-job-id-guard.sh`,
`service-registration-guard.sh`, `trade-contract-mirror-guard.sh`, and
`buff-duration-guard.sh` — this task adds no `duration` field to any
`COMMAND_TOPIC_CHARACTER_BUFF` body.

Code review before PR: `superpowers:requesting-code-review` dispatches
`plan-adherence-reviewer` and `backend-guidelines-reviewer` (Go only — no
atlas-ui files change). Pin review subagents to a cheaper model per the project's
model preference.
