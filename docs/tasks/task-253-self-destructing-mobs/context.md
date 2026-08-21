# task-253 — Implementation Context

Companion to [`plan.md`](./plan.md). Everything here was read out of the tree
while writing the plan; it exists so an implementer or reviewer does not repeat
the discovery.

---

## Key files, by service

### `atlas-data`

| File | Why it matters |
|---|---|
| `services/atlas-data/atlas.com/data/monster/reader.go:206` | `getSelfDestruction` — the absent-block sentinel. The **only** production change in this service. |
| `services/atlas-data/atlas.com/data/monster/rest.go:43` | `selfDestruction` DTO: `{action byte, remove_after int32, hp int32}`. Shape unchanged. |
| `services/atlas-data/atlas.com/data/monster/reader_test.go:1267` | The one assertion that pins the sentinel. |

### `atlas-monsters` (the bulk)

| File | Why it matters |
|---|---|
| `monster/information/model.go` | `Model` — 16 unexported fields, value receivers. `selfDestruction` is added here. |
| `monster/information/builder.go` | `ModelBuilder` exposes only what tests need; `Build()` fills a `Model` literal. |
| `monster/information/rest.go:38,56,101` | `RestModel.SelfDestruction` already exists on the DTO; `Extract` never mapped it. |
| `monster/information/processor.go` | `GetById` is Redis-cache-read-through — the threshold check costs the hot path nothing. |
| `monster/registry.go:436` | `ApplyDamage` — the `r.reg.Update` compare-and-set shape `SelfDestruct` copies. |
| `monster/registry.go:338` | `atomicUpdate` — note the comment: the mutator may run multiple times under retry, so captured flags must derive purely from `cur`. |
| `monster/processor.go:32` | `Processor` interface; `SelfDestruct` goes in the `// Commands` block after `Kill`. |
| `monster/processor.go:217` | `Create` — where the friendly/drop-period timer is armed; the self-destruct timer arm goes beside it. |
| `monster/processor.go:553` | `damageCore` — the single damage path for `Damage` **and** `Kill`. |
| `monster/processor.go:1340` | `Destroy` — unregisters the drop timer; add the self-destruct timer here too. |
| `monster/processor.go:1751` | `Kill` (Mortal Blow) — fail-closed boss guard, routes through `damageCore` with `MaxUint32`. |
| `monster/status_task.go:64` | `processDoTTick` — calls `Registry.ApplyDamage` directly, caps at `currentHp-1`. |
| `monster/drop_timer_registry.go` / `drop_timer_task.go` | The sibling pattern the self-destruct timer copies verbatim. |
| `monster/kafka.go:94,132` | `statusEventKilledBody` / `statusEventDestroyedBody`. |
| `monster/producer.go:31,149` | `destroyedStatusEventProvider` / `killedStatusEventProvider`. |
| `kafka/consumer/monster/kafka.go:13-31,137` | Command types and `killCommandBody`, with the "every handler unmarshals every message" note. |
| `kafka/consumer/monster/consumer.go:29,189` | `InitHandlers` registration list and `handleKillCommand`. |
| `main.go:65,101-109` | `InitDropTimerRegistry` and `registerSweepTasks`. |

### `atlas-channel`

| File | Why it matters |
|---|---|
| `socket/handler/monster_bomb.go` | Decode-and-log today. |
| `socket/handler/character_skill_use.go:59` | The `c.Hp() == 0` dead-character idiom. `character.Model.Hp()` is `uint16`. |
| `monster/live_mirror.go` | `GetLiveMirror().Lookup(t, uniqueId) (LiveEntry, bool)`; `LiveEntry.Field` is a `field.Model`. |
| `monster/processor.go:31,155` | `Processor` interface and `Kill`. |
| `monster/producer.go:175` | `KillCommandProvider` — the exact command-provider shape to copy. |
| `kafka/message/monster/kafka.go:20,103,181,214` | Command types and the status-event body mirrors. |
| `kafka/consumer/monster/consumer.go:191,291` | `handleStatusEventDestroyed` / `handleStatusEventKilled` and the two `*ForSession` operators that hardcode `DestroyTypeFadeOut`. |
| `main.go:917` | `handlerMap[monstersb.MonsterBombHandle]` — already routed, no wiring change needed. |

### `libs/atlas-packet`

| File | Why it matters |
|---|---|
| `monster/clientbound/destroy.go` | The codec. `swallowCharacterId` is written unconditionally on dead-type 4 today — that is the live v48–v87 desync. |
| `reactor/clientbound/spawn.go:16-31,60-70` | The canonical version-gate shape: a named predicate with its IDA sweep in the comment, tenant pulled from `ctx` inside `Encode`. |
| `test/context.go` | `test.Variants` (12 entries; v92 is index 11) and `test.CreateContext(region, major, minor)`. |
| `test/roundtrip.go:22` | `test.RoundTrip` only asserts full byte consumption — it does **not** compare struct fields. That is why the existing `TestMonsterDestroyBySwallow` survives the gate. |

### `atlas-monster-death`

| File | Why it matters |
|---|---|
| `monster/processor.go:41` | `CreateDrops` — `filterByQuestState`, `rates.GetForCharacter`, `party.GetByMemberId`, then `drop.Create` per surviving drop (return value discarded). |
| `monster/processor.go:104-110` | `filterByQuestState` — a failed quest lookup excludes **all** quest drops (fail-safe). |
| `monster/processor.go:126` | `DistributeExperience` — an error from `produceDistribution` yields a zero model whose `Solo()` map is empty, so the loop body never runs and it returns `nil`. |
| `monster/processor.go:207` | `calculateExperienceStandardDeviationThreshold` — divides by `totalEntries` and `len(ratios)`; both zero for an empty list → `NaN`. Design §8.2: pinned, not fixed. |
| `monster/drop/processor.go:87-92` | `SpawnDrop` emits a Kafka command to `drop.EnvCommandTopic` (`COMMAND_TOPIC_DROP`) — capturable with `producertest.InstallCapturing()`. |
| `quest/provider_drain_test.go:48-62` | The `httptest.NewServer` + `t.Setenv("<KEY>_SERVICE_URL", srv.URL+"/")` pattern. |

---

## Decisions this plan locks in

1. **The `atlas-data` sentinel is fixed at the source (D2), not pattern-matched
   downstream.** The plan's Task 4 presence predicate is
   `Hp > -1 || RemoveAfter > -1`, which is *also* false under the old `{0,0,0}`
   sentinel — so both rolling-deploy orders are safe, and Task 4's test table
   pins that explicitly with a "legacy absent" row.

2. **`Registry.SelfDestruct` is a new primitive, not `ApplyDamage(MaxUint32)`.**
   `ApplyDamage` reports `Killed: m.Hp() == 0`, true for every concurrent
   caller once HP hits zero. That is a latent double-kill on the ordinary
   damage path too (design §8.1) — out of scope, unchanged, and avoided here
   by construction rather than by fixing it.

3. **One kill epilogue.** Task 6 extracts `finalizeKill` from `damageCore`
   before any self-destruct code exists, so FR-6.5 ("same bookkeeping as an
   ordinary kill") is satisfied by construction and the extraction is
   reviewable as a pure refactor with the existing tests as the gate.

4. **The exported `Processor.SelfDestruct` is the single entry point** for the
   DoT, timer, and command triggers; only `damageCore` calls the unexported
   `selfDestructFrom` directly, because it already holds the block and must not
   re-fetch. This is a small divergence from design D7's sketch (which showed
   the consumer arm calling a `SelfDestruct(monsterId, characterId)` pair) —
   the added `trigger` parameter exists so the log line names the path, per the
   NFR observability requirement.

5. **A test seam for the information lookup.** `damageCore` fetched
   `information.NewProcessor(...).GetById` directly while `Kill` went through
   `testInformationLookup`. Task 7 unifies both on a new
   `p.monsterInformation(monsterId)` helper; without it, no `damageCore`
   threshold test can be written without a REST server.

6. **The channel does not check the `selfDestruction` block (D8).** Its
   monster-information client carries only `attacks`; widening it plus its
   cache to add a check atlas-monsters must repeat anyway is the wrong trade.
   PRD FR-5.3 is satisfied one hop later, by the authority.

7. **The dead-type byte passes through unmapped.** Design §2.2 swept the
   `CMobPool::Update` switch on v95 and v87 and found the meaning genuinely
   inverts between eras. Remapping would mean inventing an animation the WZ
   data did not ask for; the only real wire hazard is the dead-type-4 trailing
   field, which Task 2 gates.

---

## Dependencies and ordering

- Tasks 1, 2, 5, 6 and 13 are independent and can run in any order.
- Task 3 needs Task 2's `packet-audit:verify` markers to exist before `matrix`
  will promote anything.
- Task 4 needs Task 1 only so its "absent block" test row matches the real
  producer; the code compiles either way.
- Tasks 8, 9, 10 all need Task 7's `Processor.SelfDestruct`.
- Task 9 also patches `finalizeKill` (Task 6) and `Create`/`Destroy`.
- Task 12 needs Task 11's `monster.Processor.SelfDestruct` on the channel side.

---

## Deliberately large tasks

**Task 9 (timer)** touches 6 files across one service: two new files, the
processor, `main.go`, `registry_test.go`'s `TestMain`, and a new test file. It
is left whole because the registry, the sweep task, and the three
arm/cancel sites are one mechanism — splitting them yields a task that
lands a registry nothing writes to, which `plan-lint`'s F2 spirit (no
half-landed feature) argues against more strongly than F4's file count argues
for the split. Four of the six edits are one or two lines.

**Task 11 (channel contract)** touches 6 files, also one service, and is
similarly mechanical: three of the six are single-field or single-method
additions mirroring an existing sibling (`KillCommandBody`,
`KillCommandProvider`, `Kill`, `KillFunc`).

Everything else is at or under four files.

---

## Known risks carried from the design

| Risk | Where it is handled |
|---|---|
| PRD FR-2.1's literal `hp > -1` predicate would detonate every mob | Task 1 fixes the sentinel; `OnHpThreshold()` is the only predicate any caller uses; Task 7's test table has two no-block regression rows |
| Rolling deploy renders every death as "disappear" | Task 11's `destroyTypeFor` maps 0 → fade-out at the consumer, with a test for the absent field |
| A v83 client desyncs on dead-type 4 | Already broken today; Task 2's gate is the fix, with per-version byte fixtures in Task 2 and evidence in Task 3 |
| The Papulatus bombs (`8500003`/`8500004`) may never spawn, making the mechanic unobservable live (PRD OQ7) | The final live-observation gate names a Boomer (`5100002`) as the fallback target and requires an explicit report if the bombs do not spawn — not a silent skip |
| Timer sweep fires against a stale entry after a pod restart | Task 9's `processEntry` re-reads the monster registry and unregisters when the mob is gone or dead, exactly as `DropTimerTask.processEntry` does |

## Open item

`selfDestruction.removeAfter` is read as **seconds**, matching Cosmic
(`MapleMap.java:1868`). Design §2.4 found no client-side evidence either way —
the timer is not a client mechanic. Both in-scope timer mobs carry
`removeAfter = 0`, where the unit is unobservable, so this is a documented
convention rather than a load-bearing derivation. It re-opens only if Monster
Carnival is ever implemented (`9400547` / `9400550`).
