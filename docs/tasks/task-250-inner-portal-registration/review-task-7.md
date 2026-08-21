# Review: Task 7 — atlas-character preserves stored foothold on zero-fh movement

Commit reviewed: `b1ddb4db8` — "fix(atlas-character): preserve stored foothold on zero-fh movement commands"

Files in scope:
- `services/atlas-character/atlas.com/character/character/processor.go`
- `services/atlas-character/atlas.com/character/character/temporal_data.go`
- `services/atlas-character/atlas.com/character/character/temporal_position_test.go` (new)

## 1. The seam — every publisher of the movement command

`Move` is only called from one place in this module:
`services/atlas-character/atlas.com/character/kafka/consumer/character/consumer.go:409-416`
(`handleMovementEvent`), which consumes `character2.MovementCommand` off
`COMMAND_TOPIC_CHARACTER_MOVEMENT` (`kafka/message/character/kafka.go:402`).

Repo-wide search for producers of that topic constant:

```
services/atlas-channel/atlas.com/channel/kafka/message/movement/kafka.go   (const only)
services/atlas-parties/atlas.com/parties/kafka/consumer/character/kafka.go (const only, no Fh usage)
```

The only actual writer is `atlas-channel`'s
`movement/processor.go:ForCharacter` (line 72), which sends
`ms.Fh` — a value that has already passed through `foldMovementSummary`
(`services/atlas-channel/atlas.com/channel/movement/processor.go:283-330`).
That fold explicitly preserves the prior in-flight `Fh` unless the decoded
element's `Fh != 0` (lines 306-307, 316-317; comment at 292-294: "only when
non-zero so we don't trample the spawn-time fh ... where the client
transmits Fh=0 for 'no anchor yet'"). Confirmed via `git log -p --follow`
that `foldMovementSummary` and this comment predate task-250 entirely
(present since `207361fdd` "Perform all movement processing in channel" and
carried through unrelated commits — not touched on this branch).

`ForNPC`/`ForMonster` publish to different topics
(`EnvCommandPetMovement`, `EnvCommandMonsterMovement`), so they are
irrelevant to `character.Move`.

Conclusion: no existing publisher of `COMMAND_TOPIC_CHARACTER_MOVEMENT`
today relies on `Fh == 0` meaning "foothold zero" — the producer-side fold
already treats 0 as "unknown/preserve" before it ever reaches Kafka. This
change is consistent with the one real producer's existing semantics, not
just the not-yet-implemented inner-portal path. **PASS.**

## 2. `UpdatePosition` signature change — "no callers" claim

Verified independently against the pre-commit tree
(`git grep -n UpdatePosition 5f299e4bb`): the only match in
`services/atlas-character` is the function's own definition at
`temporal_data.go:73`. The other `UpdatePosition` hits are an unrelated type
in `services/atlas-pets` (`pet.TemporalRegistry.UpdatePosition`, different
package, different signature, not part of this module). The brief's premise
holds and the signature change is safe. **PASS.**

## 3. Test quality

All three specified tests exist in `temporal_position_test.go` and assert
exactly the brief's table (x/y/fh/stance via `GetById(...).X()/.Y()/.Fh()/.Stance()`).

- `TestMove_ZeroFh_PreservesStoredFoothold` — seeds `fh=77` via `Update`,
  calls `Move(..., fh=0, ...)`, asserts `fh==77` survives. Against the
  pre-change `Move` (which called `Update` unconditionally with `fh=0`),
  this fails — confirmed by the report's captured RED run
  (`got x=300 y=-50 fh=0 stance=5, want ... fh=77 ...`) and independently
  reproduced by running the current suite (all three pass, see below).
- `TestMove_NonZeroFh_OverwritesFoothold` — control case, passes both
  before and after (expected; it is not the regression case, it is a
  guard against a naive "always preserve" fix).
- `TestMove_ZeroFh_NoPriorState_StoresZeroFh` — covers the `existing`
  zero-value read-miss path; also passes both before and after, but is a
  legitimate boundary case (no prior redis entry) worth pinning.

Ran `go test ./character/... -run 'TestMove_' -v` in the module: all three
`PASS`. The `redis: connection pool: failed to dial` stderr lines are
`miniredis` teardown noise from the shared `sync.Once` registry, not test
failures — consistent with the report's explanation and reproduced
independently.

Tenant isolation: `setupMoveTest` calls `tenant.Create(uuid.New(), ...)` per
test and `GetTemporalRegistry()` is a process-wide `sync.Once` singleton
(`temporal_data.go:105-121`) reused across tests via
`InitTemporalRegistry`/`setupResourceTestRegistry` (`resource_test.go:31-36`).
Because the registry keys reads/writes by `(tenant, characterId)`
(`reg.Get(ctx, t, characterId)`), a fresh tenant per test prevents
cross-test bleed even though the underlying miniredis client is shared.
Character ID 42 is reused across the first two tests but under distinct
tenants, so no collision. **PASS.**

## 4. Conventions

- Builder pattern / test setup: reuses the existing `setupResourceTestRegistry`
  helper from `resource_test.go` (same package) rather than inventing a new
  helper file. No new `*_testhelpers.go` was added (`git diff --stat` shows
  only `temporal_position_test.go` as new). **PASS.**
- No new domain constant introduced; `fh == 0` sentinel is a pre-existing
  domain convention already encoded on the `atlas-channel` producer side,
  not a new magic number invented here. **PASS.**
- `temporalData` immutability preserved: both `Update` and the modified
  `UpdatePosition` still construct a fresh `temporalData{...}` value and
  `Put` it wholesale; no in-place mutation of the struct was introduced.
  **PASS.**
- Comment on `UpdatePosition` (`temporal_data.go:73-77`) matches the
  density/idiom of the file (doc comments elsewhere in this package are
  similarly terse); it correctly cites the mirrored channel-side rule and
  file. **PASS.**

## 5. Minor/non-blocking observations

- `Move`'s two-branch shape (`if fh == 0 { ...; return nil }` then fall
  through to `Update`) is a reasonable minimal diff, but note it duplicates
  the `tenant.MustFromContext(p.ctx)` computation into a local `t` used by
  both branches — fine, not a defect, just noting the diff is tightly
  scoped as intended by the brief (no test added for `Move`'s error paths,
  but `Move` has no error paths to test — both registry calls are
  fire-and-forget writes, matching pre-existing behavior).
- The doc comment repeats "atlas-channel movement/processor.go" without a
  line number; harmless since the file was actively refactored around that
  logic and a specific line reference would risk rotting, but call it out
  as a style note only.

## Verdict rationale

All five checklist items pass with direct evidence. The cross-service seam
was traced to its one real producer and the fold rule was confirmed
pre-existing (not something task-250 quietly introduced elsewhere to make
this consumer-side change look safer than it is). The "no callers" premise
for the signature change was verified independently against the pre-commit
tree. All three specified tests exist, assert the brief's table, and the
regression-pinning test was confirmed to fail pre-change via the report's
captured run plus my own reproduction of the current green suite.
