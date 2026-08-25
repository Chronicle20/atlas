# Fix report — contact-trigger self-destructs use DeathTypeBomb

Task: task-253-self-destructing-mobs
Brief: `docs/tasks/task-253-self-destructing-mobs/bug-darkstar-no-explosion-or-damage.md`
("Fix — DECIDED 2026-08-25 (reporter approved)")

## What I implemented

1. **`services/atlas-monsters/atlas.com/monsters/monster/processor.go`**
   - `ProcessorImpl.SelfDestruct` (was at line 1847): when `trigger == TriggerContact`, the
     resolved `deathType` is forced to `DeathTypeBomb` (the existing `"BOMB"` constant at
     `kafka.go:69`) instead of `deathTypeForAction(sd.Action())`. `TriggerThreshold` and
     `TriggerTimer` are untouched and keep the WZ-derived deathType (confirmed: the threshold
     path at `processor.go:695` calls `deathTypeForAction` directly and was not touched; the
     timer path at `self_destruct_timer_task.go:46` calls `SelfDestruct(uid, 0, TriggerTimer)`
     and is unaffected by the new `if`).
   - Every existing guard (unknown monster, already dead, information-lookup failure, absent
     `selfDestruction` block) and the `selfDestructFrom` exactly-once epilogue are untouched —
     the new branch sits strictly between resolving `deathType` and calling
     `selfDestructFrom`.
   - Updated the `SelfDestruct` doc comment to state the TriggerContact deviation and cite the
     bug file, quoting the v83 `CMob::OnDie` dead-type-3 divergence and the
     dead-type-2/4/5-only reachability of `CMob::OnBomb` (the explode action that also carries
     the player-damaging attack), per ruling 4.
   - Updated the `deathTypeForAction` doc comment to note that `SelfDestruct`'s TriggerContact
     path does not consume its result, with a cross-reference to the bug file.
   - No literal byte was written anywhere — `DeathTypeBomb` is the existing string-key
     constant already resolved through the tenant `operations` table (DOM-25). Confirmed
     `BOMB` is not new: `grep -rl "DeathTypeBomb\|\"BOMB\""` only turns up `kafka.go` (the
     existing constant declaration) and `processor.go` (now two use sites); I added no new
     seed-template plumbing per ruling 2 ("verify, don't add").

2. **`services/atlas-monsters/atlas.com/monsters/monster/self_destruct_test.go`**
   - `TestSelfDestructRejects`'s "valid target" subtest (contact trigger, WZ action 3) now
     asserts `DeathType == DeathTypeBomb` instead of `DeathTypeDestructByMiss`, with a comment
     explaining why (it exercises `TriggerContact`, which now always emits `BOMB`).
   - Added `TestSelfDestructContactAlwaysBomb`, a table test covering all four rows the brief's
     "Tests" section asks for: contact with WZ action 1 → `BOMB`, contact with WZ action 3 →
     `BOMB`, threshold with WZ action 3 → `DESTRUCT_BY_MISS` (WZ pass-through unchanged),
     timer with WZ action 3 → `DESTRUCT_BY_MISS` (WZ pass-through unchanged). Built with the
     project's existing `information.NewModelBuilder()` / `field.NewBuilder()` /
     `r.CreateMonster` Builder pattern already used by the surrounding tests in this file — no
     new test-only constructors.

3. **`docs/tasks/task-253-self-destructing-mobs/design.md`** (§2.2 correction, ruling 5)
   - Added the v83 `CMobPool::Update` (`754107bf`, `0x679138`) two-arm switch alongside the
     existing v87 arm — it has the identical `{0,1,3}` → `OnDie` / `{2,4,5}` → bomb shape, with
     `OnDie`/`OnBomb` addresses `0x663995`/`0x663e5b`.
   - Added a correction paragraph: dead-type 3 routing to `OnDie` is not "an ordinary death" —
     `OnDie` itself branches on `m_nDeadType == 3` and picks a dedicated one-time action (21 on
     v83, 22 on v95) with no `die1..dieN` fallback, quoting the same v83 `0x663a1b` / v95
     `0x64e6bc` disassembly the bug file cites. Did **not** rewrite the original v87/v95
     paragraph — the correction is appended as a distinct paragraph immediately after it, per
     ruling 5 ("do not silently rewrite").
   - Added an "Amendment" paragraph at the end of D2 (not a rewrite of D2's original text)
     recording the new TriggerContact→`DeathTypeBomb` deviation, why it's needed (the §2.2
     correction), why `DeathTypeBomb`/byte 2 rather than Cosmic's literal 4 (v92+/JMS swallow-
     arm hazard), and that `TriggerThreshold`/`TriggerTimer` are unaffected.

4. **`docs/tasks/task-253-self-destructing-mobs/prd.md`** (ruling 6)
   - §6.3 table row: `| 5100002 | 1 | 1800 | — | HP | Boomer |` → `... | Firebomb |`.
   - §3 user story (`prd.md:66`): "As a player killing a Boomer..." → "As a player killing a
     Firebomb...".
   - Left `prd.md:21`, `:110`, `:438` (other "Boomer" mentions) untouched — the brief scoped
     the fix to "§6.3's row annotation and the prd.md:66 user story" only; widening further
     was out of scope for this task.

## Scope limits honored

- No change to Firebomb/threshold behavior (`processor.go:695`, the `damageCore` threshold
  arm, is untouched).
- No server-side explosion damage added — `selfDestructFrom` still only does registry
  transition, credit, and `finalizeKill`; no new damage emission.
- `getFirstAttack` / `atlas-data` untouched.

## Tests

Command (module-local, from the `atlas-monsters` module root):

```
cd services/atlas-monsters/atlas.com/monsters && go build ./... && go test ./monster/ -run 'TestSelfDestruct' -v
```

Output (tail):

```
=== RUN   TestSelfDestructRejects
...
--- PASS: TestSelfDestructRejects (0.01s)
    --- PASS: TestSelfDestructRejects/unknown_monster (0.00s)
    --- PASS: TestSelfDestructRejects/already_dead (0.00s)
    --- PASS: TestSelfDestructRejects/no_selfDestruction_block (0.00s)
    --- PASS: TestSelfDestructRejects/information_lookup_fails (0.00s)
    --- PASS: TestSelfDestructRejects/valid_target (0.00s)
=== RUN   TestSelfDestructContactAlwaysBomb
=== RUN   TestSelfDestructContactAlwaysBomb/contact_with_WZ_action_1_(fadeOut)_resolves_to_bomb
=== RUN   TestSelfDestructContactAlwaysBomb/contact_with_WZ_action_3_(destructByMiss)_resolves_to_bomb
=== RUN   TestSelfDestructContactAlwaysBomb/threshold_keeps_WZ-derived_deathType
=== RUN   TestSelfDestructContactAlwaysBomb/timer_keeps_WZ-derived_deathType
--- PASS: TestSelfDestructContactAlwaysBomb (0.01s)
    --- PASS: TestSelfDestructContactAlwaysBomb/contact_with_WZ_action_1_(fadeOut)_resolves_to_bomb (0.00s)
    --- PASS: TestSelfDestructContactAlwaysBomb/contact_with_WZ_action_3_(destructByMiss)_resolves_to_bomb (0.00s)
    --- PASS: TestSelfDestructContactAlwaysBomb/threshold_keeps_WZ-derived_deathType (0.00s)
    --- PASS: TestSelfDestructContactAlwaysBomb/timer_keeps_WZ-derived_deathType (0.00s)
=== RUN   TestSelfDestructAttributesToDamageLeader
--- PASS: TestSelfDestructAttributesToDamageLeader (0.00s)
=== RUN   TestSelfDestructNoDamageEntriesReportsNoKiller
--- PASS: TestSelfDestructNoDamageEntriesReportsNoKiller (0.00s)
=== RUN   TestSelfDestructIsIdempotent
--- PASS: TestSelfDestructIsIdempotent (0.00s)
=== RUN   TestSelfDestructTimerTaskFiresOnElapsedEntry
--- PASS: TestSelfDestructTimerTaskFiresOnElapsedEntry (0.00s)
=== RUN   TestSelfDestructTimerTaskSkipsUnelapsedEntry
--- PASS: TestSelfDestructTimerTaskSkipsUnelapsedEntry (0.00s)
=== RUN   TestSelfDestructTimerTaskUnregistersDeadMob
--- PASS: TestSelfDestructTimerTaskUnregistersDeadMob (0.00s)
=== RUN   TestSelfDestructTimerTaskUnregistersMissingMob
--- PASS: TestSelfDestructTimerTaskUnregistersMissingMob (0.00s)
PASS
ok  	atlas-monsters/monster	0.048s
```

Full module build and package test suite also run clean:

```
go build ./...
go test ./monster/...
ok  	atlas-monsters/monster	17.320s
ok  	atlas-monsters/monster/consumable	0.019s
?   	atlas-monsters/monster/consumable/mock	[no test files]
ok  	atlas-monsters/monster/drop	0.024s
?   	atlas-monsters/monster/drop/mock	[no test files]
ok  	atlas-monsters/monster/information	15.513s
?   	atlas-monsters/monster/information/mock	[no test files]
?   	atlas-monsters/monster/mobskill	[no test files]
?   	atlas-monsters/monster/mobskill/mock	[no test files]
```

Output is pristine (no warnings beyond the expected kafka-writer info/warn lines the timer-task
tests already log, unrelated to this change).

## Files changed

- `services/atlas-monsters/atlas.com/monsters/monster/processor.go`
- `services/atlas-monsters/atlas.com/monsters/monster/self_destruct_test.go`
- `docs/tasks/task-253-self-destructing-mobs/design.md`
- `docs/tasks/task-253-self-destructing-mobs/prd.md`
- `docs/tasks/task-253-self-destructing-mobs/fix-contact-bomb-report.md` (this report)

## Self-review

- Confirmed the contact command consumer (`kafka/consumer/monster/consumer.go:206`) calls
  `monster.NewProcessor(l, ctx).SelfDestruct(c.MonsterId, c.Body.CharacterId,
  monster.TriggerContact)` — no consumer-level test asserts `deathType` directly, so no change
  needed there; the processor-level tests are the correct place per the brief's "Tests"
  section.
- Confirmed the threshold path (`processor.go:695`, inside `damageCore`) calls
  `deathTypeForAction(p.l, sd.Action())` directly, not through `SelfDestruct`, so it was
  already immune to the `TriggerContact` branch and needed no change.
- Confirmed `DeathTypeBomb` was not newly introduced — it already existed at `kafka.go:69` and
  is used as the resolved key for wire byte 2 through the existing `operations` table
  mechanism; grepped for any other `DeathTypeBomb`/`"BOMB"` occurrence across the repo and
  found none outside `atlas-monsters` and this bug file, consistent with "verify, don't add."
  I did not touch any tenant seed-template data, and did not touch `getFirstAttack` in
  `atlas-data` (out of this task's scope per the brief).
- Verified `design.md`'s new correction paragraph is additive (appended, not a rewrite) both
  for §2.2 and for D2, per ruling 5's "do not silently rewrite D2's original text."
- Verified `prd.md`'s other "Boomer" mentions (`:21`, `:110`, `:438`) were left alone —
  intentionally out of the ruling-6 scope.

## Issues or concerns

None. The change is a minimal, additive branch in `SelfDestruct` scoped exactly to
`TriggerContact`, backed by table-driven tests covering all three triggers, and the two doc
corrections are additive per the rulings.
