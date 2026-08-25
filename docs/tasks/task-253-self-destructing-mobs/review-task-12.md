# Review: Task 12 — atlas-channel MONSTER_BOMB handler

Range reviewed: `61457cf1a..79b5f3bbf` (single commit `79b5f3bbf`).
Brief: `.superpowers/sdd/plan/task-12-brief.md`
Report: `.superpowers/sdd/plan/task-12-report.md`

## Scope

`git diff --stat 61457cf1a..79b5f3bbf`:

```
services/.../socket/handler/monster_bomb.go      | 55 +++++-
services/.../socket/handler/monster_bomb_test.go | 213 +++++++++++++++++++++
2 files changed, 267 insertions(+), 1 deletion(-)
```

Matches the brief's file list exactly. No codec file touched (`libs/atlas-packet/monster/serverbound/monster_bomb.go` is unchanged, confirmed by `git diff` showing no hunks for it). Task 11's `monster.Processor.SelfDestruct` and `monster.GetLiveMirror()` are consumed read-only, as specified.

## Findings

### 1. Rejection/detonation logic — PASS

`socket/handler/monster_bomb.go:38-66` (post-change) implements, in order:

1. Resolve character via `monsterBombCharacterFunc` — error → log `unable to resolve`, return (no detonation). Verified by `TestMonsterBombRejects/character_lookup_fails`.
2. `c.Hp() == 0` → log `while dead`, return. Verified by `TestMonsterBombRejects/dead_character`. Idiom matches `character_skill_use.go:59`'s `c.Hp() == 0` pattern as instructed.
3. `monster.GetLiveMirror().Lookup(...)` miss → log `not in the live mirror`, return. Verified by `TestMonsterBombRejects/mirror_miss`.
4. `entry.Field.Id() != s.Field().Id()` → log `not in the reporter's field`, return. Verified by `TestMonsterBombRejects/mob_in_another_field`.
5. Only past all four guards does `monsterBombSelfDestructFunc` fire, verified positively by `TestMonsterBombDetonates` (called exactly once with the reporter's field/monsterId/characterId) and negatively by all four rejection subtests asserting `selfDestructCalls != 0 → Fatalf`.

Ran independently: `go test ./socket/handler/ -run MonsterBomb -v -race` — all 7 cases (3 top-level + 4 table rows) PASS, no race flagged.

### 2. Package-level seam restoration — PASS, no cross-test leakage

Both `monsterBombCharacterFunc` and `monsterBombSelfDestructFunc` are swapped in `newMonsterBombEnv` (`monster_bomb_test.go:76-92`) and restored via `t.Cleanup(func() { monsterBombCharacterFunc = origChar })` / `t.Cleanup(func() { monsterBombSelfDestructFunc = origSelfDestruct })`. `t.Cleanup` is registered per-call to `newMonsterBombEnv`, which every test (including every `TestMonsterBombRejects` subtest via `t.Run`) invokes once, so each subtest gets its own capture/restore pair, executed in LIFO order after that subtest returns — functionally equivalent to a `defer` in the brief's spec (and, since `t.Cleanup` runs even after `t.Fatalf`'s `runtime.Goexit()`, at least as safe). Confirmed no leakage by running the full suite with `-race`: no data race, and sequential execution of `TestMonsterBombRejects`'s four subtests each independently pass/fail on their own case only.

Session registry is also cleaned per test: `session.AddSessionToRegistry` is paired with `t.Cleanup(func() { session.ClearRegistryForTenant(ten.Id()) })` (`monster_bomb_test.go:57-58`).

The live mirror (`monster.GetLiveMirror()`) is a process-wide singleton keyed by tenant UUID, and is *not* explicitly cleaned up after each test's `Put` call. This is not a leakage bug: `mustTenant` (`character_cash_item_use_test.go:23-30`) mints a fresh `uuid.New()` tenant id on every call, so each test/subtest owns a disjoint keyspace in the mirror — no cross-test interference, just an unbounded (but harmless, test-process-lifetime) memory growth. Noted as non-blocking.

### 3. Tests drive a real encoded packet — PASS

`monster_bomb_test.go:24-26` builds the wire bytes directly (`{0x59, 0x1B, 0x00, 0x00}`, little-endian mob id 7001), matching the codec at `libs/atlas-packet/monster/serverbound/monster_bomb.go:47` (`Encode4`/`ReadUint32`, single field). `dispatch()` (`monster_bomb_test.go:99-104`) wraps this in a real `request.Request` + `request.NewRequestReader` and calls the exported `MonsterBombHandleFunc(...)` — no internal helper is called directly, no `serverbound.MonsterBomb{mobId: ...}` struct-literal shortcut (the type's field is unexported anyway, so that shortcut isn't even available from this package). Verified the byte encoding is correct: 7001 = 0x1B59, LE bytes `59 1B 00 00` — matches.

### 4. `SetField` fixture note — PASS, accurate, test not passing for the wrong reason

Claim in `monster_bomb_test.go:107-112`: `session.ProcessorImpl.SetField` only mutates `mapId`/`instance`, not `worldId`/`channelId`.

Verified directly: `session/processor.go:303-313` (`SetField`) calls `s.setMapId(f.MapId())` and `s.setInstance(f.Instance())` only — no `setWorldId`/`setChannelId` calls (those exist as separate unexported setters, `session/model.go:155-166`, but `SetField` never invokes them).

Verified the workaround is correct, not accidentally-passing: `session.NewSession` (`session/model.go:42`) leaves `field` at its Go zero value, i.e. `world.Id(0)`, `channel.Id(0)` (`world.Id`/`channel.Id` are `byte`-typed, confirmed in `libs/atlas-constants/world/constants.go:3` and `libs/atlas-constants/channel/constants.go:3`, so zero value is `0`). The test's `monsterBombField()` (`monster_bomb_test.go:105-113`) explicitly builds with `world.Id(0)` and channel `0`, matching the session's zero-value default. `field.Model.Id()` (`libs/atlas-constants/field/model.go:21-23`) is a formatted string of all four components (worldId, channelId, mapId, instance), so had the test instead used a non-zero world/channel (as the brief's own field example `(0, 1, 100000000)` literally specifies — channel 1, not 0), `entry.Field.Id()` (test-built, channel 1) would never equal `s.Field().Id()` (session default, channel 0 post-`SetField`), and `TestMonsterBombDetonates` would fail on the "not in the reporter's field" guard instead of reaching the seam. The implementer's fix (using channel `0` throughout, deviating from the brief's literal `(0, 1, ...)` example) is necessary and correctly documented; the test genuinely exercises the field-match guard rather than accidentally passing.

One minor note: this means `TestMonsterBombRejects`'s "mob in another field" case is the only place a non-zero-vs-default field mismatch is actually exercised (map 100000000 vs 200000000, both channel/world 0) — it does correctly test the field-mismatch path, just never varies world/channel, only mapId. This is sufficient to cover the `Id()` comparison guard as implemented and is not a gap given the brief's own scope (field mismatch by map is the realistic case; a channel-only mismatch would be an unusual routing bug outside this handler's stated guards).

### 5. Deferred-marker removal, gate-lint — PASS

`grep -rn "behavior: deferred" socket/handler/monster_bomb.go` → no output (confirmed independently). No new codec touched, no new raw-comparison site introduced — the 38 pre-existing gate-lint sites are unaffected, out of scope per standing ruling.

### 6. Build/test — PASS

`go build ./...` and `go test ./socket/handler/ -run MonsterBomb -v -race` both green, confirmed independently (not just taken from the report).

## Not evaluable

- Whether `monster.Processor.SelfDestruct`'s Kafka emission is itself correct is Task 11's surface, not this handler's — read only far enough to confirm the call signature matches (`monster/processor.go:162`).

## Verdict

APPROVED — all four brief-mandated guard behaviors verified independently against real evidence, seam restoration confirmed leak-free, tests confirmed to drive a real encoded packet through the exported handler, and the `SetField` fixture note confirmed accurate and load-bearing (not a coincidental pass).
