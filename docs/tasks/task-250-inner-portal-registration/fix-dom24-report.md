# DOM-24 fix report — movement package producer stub

## Task

Scoped fix for the single blocking backend-guidelines review finding (DOM-24):
`movement/teleport_test.go`'s three earlier tests exercised
`ProcessorImpl.TeleportCharacter` / `ForCharacter`, which reach the real
un-stubbed `producer.ProviderImpl` (`movement/processor.go:97`) asynchronously
via `routine.Go`, because the `movement` package had no `TestMain` installing a
producer stub.

## What I implemented

1. **`services/atlas-channel/atlas.com/channel/movement/testmain_test.go`** (new) —
   package-level `TestMain` that installs a producer stub once, before any test
   (and therefore any async goroutine) runs.

2. **`services/atlas-channel/atlas.com/channel/movement/teleport_test.go`** — changed.
   See "Why I changed teleport_test.go" below.

### Why I changed teleport_test.go

The brief's literal instruction was `producertest.InstallNoop()` in `TestMain`
(the berserk precedent), leaving `TestTeleportCharacter_EmitsFhZeroOnWire`'s
existing `producertest.InstallCapturing()` / `t.Cleanup(producertest.InstallNoop)`
pair untouched. I implemented that literally first and verified it under
`-race`. It reproducibly failed:

```
WARNING: DATA RACE
Write at 0x00000134e3d8 by goroutine 79:
  .../producer.ResetInstance()
      .../libs/atlas-kafka/producer/manager.go:44
  .../producertest.InstallCapturing()
      .../teleport_test.go:110
  atlas-channel/movement.TestTeleportCharacter_EmitsFhZeroOnWire()

Previous read at 0x00000134e3d8 by goroutine 71:
  sync/atomic.LoadUint32() ... (inside sync.Once.Do, via GetManager)
  github.com/Chronicle20/atlas/libs/atlas-routine.Go.func1()
      .../libs/atlas-routine/routine.go:24

Goroutine 71 created at:
  atlas-channel/movement.(*ProcessorImpl).TeleportCharacter()
      .../movement/processor.go:97
  atlas-channel/movement.TestTeleportCharacter_NoClientboundBroadcast()
      .../teleport_test.go:56
```

Root cause: `producer.ResetInstance()` does a plain (non-atomic) reassignment
of the package-level `sync.Once` (`libs/atlas-kafka/producer/manager.go:44`),
which `sync.Once.Do`'s fast path reads via an atomic load. `InstallCapturing`
(like `InstallNoop`) calls `ResetInstance` internally. `TeleportCharacter`'s
Kafka emit is fire-and-forget (`routine.Go`, never awaited by the caller), so
an earlier test's async goroutine can still be reading the singleton via
`producer.ProviderImpl` → `GetManager` when a later test calls
`InstallCapturing()` again — a genuine data race, not an artifact of my
`TestMain` addition.

I confirmed this is pre-existing and independent of my change: temporarily
removing `testmain_test.go` and rerunning `go test ./movement/... -race
-run 'TestTeleportCharacter|TestForCharacter'` reproduced the identical race
(same call sites, same goroutines) against the original, `TestMain`-less code.
`teleport_test.go` was added entirely within this task branch (commits
`a1fa2fcfb`, `bb38fa567`), not inherited from `main`, so this is in-scope
reconciliation of the very file the brief asked me to touch, not a
new/unrelated finding.

**Fix**: install `producertest.InstallCapturing()` once in `TestMain` (instead
of `InstallNoop`) and store the returned `*producertest.Capture` in a
package-level `sharedCapture` var. `ResetInstance` is then called exactly
once, before any test (and therefore any goroutine) exists, which eliminates
the race by construction — there is no second call to reset the singleton
anywhere in the package. `TestTeleportCharacter_EmitsFhZeroOnWire` no longer
calls `InstallCapturing`/`t.Cleanup(InstallNoop)`; it calls
`sharedCapture.Reset()` (which only clears previously recorded messages under
a mutex — it does not touch the singleton) and reads via `sharedCapture`. The
other three tests are unaffected: `CapturingWriter.WriteMessages` discards
nothing to a broker, same as `NoopWriter`, so switching the package default
from Noop to Capturing is behaviorally transparent to tests that don't
inspect what was produced.

This diverges from the brief's literal `InstallNoop()` instruction in
`testmain_test.go`, per the brief's own "if it conflicts, reconcile so both
hold" clause — the literal instruction and a race-clean `-race` run are
mutually exclusive here, and I chose the reconciliation that keeps the
`-race` run clean without weakening `TestTeleportCharacter_EmitsFhZeroOnWire`'s
assertions (it still verifies `Fh == 0`, `X`/`Y`, and `ObjectId` on the
captured wire message).

## Evidence

`go build ./...` — clean, no output.

`go test ./movement/... -race -count=1` — ran 5 times back-to-back, all pass:

```
ok  	atlas-channel/movement	1.101s
ok  	atlas-channel/movement	1.071s
ok  	atlas-channel/movement	1.088s
ok  	atlas-channel/movement	1.048s
ok  	atlas-channel/movement	1.049s
```

Full verbose run (`go test ./movement/... -v`) — all 27 tests pass, including
the four `teleport_test.go` tests:

```
=== RUN   TestTeleportCharacter_WritesLastPosition
--- PASS: TestTeleportCharacter_WritesLastPosition (0.00s)
=== RUN   TestTeleportCharacter_NoClientboundBroadcast
--- PASS: TestTeleportCharacter_NoClientboundBroadcast (0.00s)
=== RUN   TestForCharacter_WritesLastPosition
--- PASS: TestForCharacter_WritesLastPosition (0.00s)
=== RUN   TestTeleportCharacter_EmitsFhZeroOnWire
--- PASS: TestTeleportCharacter_EmitsFhZeroOnWire (0.00s)
PASS
```

Module-local full suite from `services/atlas-channel/atlas.com/channel`:

```
$ go build ./...
$ go test ./...
```

No `FAIL` anywhere in the output (`grep -E 'FAIL|movement'` on the full run
output showed only `ok atlas-channel/movement (cached)` and two
`[no test files]` lines for unrelated `movement`-prefixed packages).

## Files changed

- `services/atlas-channel/atlas.com/channel/movement/testmain_test.go` (new)
- `services/atlas-channel/atlas.com/channel/movement/teleport_test.go` (edited — see above)

## Self-review

- Completeness: the async-reaches-real-producer defect (DOM-24) is fixed for
  all three previously-unstubbed tests, and the fourth test's existing
  assertions are unchanged (same Fh/X/Y/ObjectId checks).
- Discipline: no production code touched (`movement/processor.go` untouched);
  fix is entirely test-scaffolding. No unrelated audit findings acted on.
- Testing: verified both the originally-specified literal fix (InstallNoop,
  unchanged capturing test) and the final fix under `-race`, and documented
  why the first was rejected.

## Concerns

None blocking. Noting for the record: `TeleportCharacter`/`ForCharacter`'s
Kafka emit remains genuinely fire-and-forget in production code
(`movement/processor.go`) — that's out of scope for DOM-24, which is a
test-only finding, but it's the underlying reason this package's tests must
stay careful about producer-stub install timing.
