# Review: Task 6 — `movement.TeleportCharacter` and the last-position write

Commit reviewed: `a1fa2fcfb` (range `d5433181a..a1fa2fcfb`)

Files in scope:
- `services/atlas-channel/atlas.com/channel/movement/processor.go`
- `services/atlas-channel/atlas.com/channel/movement/mock/processor.go`
- `services/atlas-channel/atlas.com/channel/movement/teleport_test.go` (new)

Note per instructions: `TeleportCharacter` has no production caller in this
commit by design — the inner-portal handler task consumes it later. Not
flagged as dead code.

## 1. The cross-service seam — `fh` into `atlas-character.Move`

Traced by hand, not by inspection alone:

- `movement/processor.go:93-101` — `TeleportCharacter` calls
  `CommandProducer(f, uint64(characterId), characterId, x, y, 0, 0)` — `fh`
  and `stance` both hard-coded `0`.
- `movement/producer.go:13-29` — `CommandProducer` maps that `fh` argument
  straight into `movement.Command[any]{... Fh: fh ...}`, published to
  `EnvCommandCharacterMovement` (`COMMAND_TOPIC_CHARACTER_MOVEMENT`).
- `atlas-character/kafka/consumer/character/consumer.go:409-411` —
  `handleMovementEvent` unmarshals `character2.MovementCommand` and calls
  `character.NewProcessor(...).Move(uint32(c.ObjectId), c.X, c.Y, c.Fh, c.Stance)`
  — `c.Fh` passed through unchanged.
- `atlas-character/character/processor.go:876-884` (task 7, commit
  `b1ddb4db8`, already on branch) — `Move`: `if fh == 0 {
  GetTemporalRegistry().UpdatePosition(...) ; return nil }` else
  `Update(...)`.

`TeleportCharacter`'s `fh=0` therefore lands on `UpdatePosition`, which
preserves the character's stored foothold rather than overwriting it with the
inner-portal's fabricated zero. This is exactly the interaction the brief's
doc comment and task 7's fix describe. Confirmed by reading the actual
consumer code, not by trusting the comment. **No mismatch found.**

`ForCharacter`'s unaffected publish path (line 86) still sends the folded
`ms.Fh`, which is `0` only when `foldMovementSummary` decided there is no
real foothold in-flight (pre-existing behavior on this branch, not touched
here) — also routes correctly to the same preserving branch. Consistent.

## 2. Registry write in `ForCharacter` (`processor.go:84`)

```go
position.GetRegistry().Put(p.t, characterId, position.Position{X: ms.X, Y: ms.Y})
err = producer.ProviderImpl(...)(CommandProducer(f, uint64(characterId), characterId, ms.X, ms.Y, ms.Fh, ms.Stance))
```

- Lives inside the second `routine.Go`, **after** the fold succeeds (`ms, err
  := model2.Fold(...); if err != nil { return }`) and **before** the publish
  — matches the brief exactly (`processor.go:76-86`).
- On fold error, the function returns before reaching `Put`, so a failed fold
  cannot write a stale/zero/garbage position. Correct.
- `p.t` is the tenant captured in `NewProcessor` via
  `tenant.MustFromContext(ctx)` (`processor.go:62`) — the registry key is
  `{Tenant, CharacterId}` (`position/registry.go:20-23`), so writes are
  tenant- and character-scoped correctly; no cross-tenant/cross-character
  leak. Confirmed by reading `position/registry.go` (pre-existing, read-only
  per the brief).
- No other write path was added or altered in this commit that could race or
  overwrite this value out of order (the `ForNPC`/`ForPet`/`ForMonster`
  writes are unrelated registries — `monster.GetLiveMirror()` etc. — not
  touched here).

## 3. `TeleportCharacter` implementation

`movement/processor.go:92-101`:

```go
func (p *ProcessorImpl) TeleportCharacter(f field.Model, characterId uint32, x int16, y int16) error {
	position.GetRegistry().Put(p.t, characterId, position.Position{X: x, Y: y})
	routine.Go(p.l, p.ctx, func(_ context.Context) {
		err := producer.ProviderImpl(p.l)(p.ctx)(movement2.EnvCommandCharacterMovement)(CommandProducer(f, uint64(characterId), characterId, x, y, 0, 0))
		if err != nil {
			p.l.WithError(err).Errorf("Unable to issue movement command [%d].", characterId)
		}
	})
	return nil
}
```

- Registry write is synchronous (before `routine.Go`), matching the brief's
  ordering requirement so a synchronous `Lookup` right after the call
  observes it (`EnterInner`'s later plausibility check needs this — not part
  of this task's scope, but the ordering is what makes it possible).
- Publish uses the same `CommandProducer` shape and same
  `EnvCommandCharacterMovement` env key as `ForCharacter`, same error-log
  format (`"Unable to issue movement command [%d]."`) — consistent with the
  file's existing convention.
- No clientbound announce anywhere in the method — satisfies "emits NO
  clientbound broadcast" directly (there is nothing to suppress; the method
  simply never calls `session.Announce`/`ForOtherSessionsInMap`).
- Doc comment on the interface (`processor.go:43-55`) matches the brief's
  text verbatim, including the `fh=0` rationale and cross-service pointer.

## 4. Mock (`movement/mock/processor.go`)

```go
TeleportCharacterFunc func(f field.Model, characterId uint32, x int16, y int16) error
...
func (m *ProcessorMock) TeleportCharacter(f field.Model, characterId uint32, x int16, y int16) error {
	if m.TeleportCharacterFunc != nil {
		return m.TeleportCharacterFunc(f, characterId, x, y)
	}
	return nil
}
```

Matches the file's existing four fields/methods exactly: same nil-guard
pattern, same field-naming convention (`<Method>Func`), gofmt-aligned struct
tags, method placed after the other four in file order, `var _
movement.Processor = (*ProcessorMock)(nil)` compile-time check still holds
(verified via `go build ./movement/...` — no output, success).

## 5. Test honesty (`movement/teleport_test.go`)

Three tests, all reusing `newMovementTestTenant`/`newMovementTestProcessor`/
`movementTestField` from `movement/processor_test.go` (same package) rather
than redefining helpers — matches the brief.

- `TestTeleportCharacter_WritesLastPosition` — calls `TeleportCharacter(f,
  42, 300, -50)`, asserts `position.GetRegistry().Lookup(tm, 42)` returns
  `{300, -50}, true`. This fails without the `Put` call added in this
  commit — real assertion against the actual singleton registry, not a mock.
- `TestTeleportCharacter_NoClientboundBroadcast` — real `writer.Producer`
  spy counting invocations, asserts `announces == 0`. Genuinely exercises the
  "no broadcast" contract; would fail if a future change added an announce
  call to this method.
- `TestForCharacter_WritesLastPosition` — builds a `packetmodel.Movement`
  that folds to `(150, 250)`, calls `ForCharacter`, polls
  `position.GetRegistry().Lookup` with a 1s bound (correctly justified: the
  write happens inside `ForCharacter`'s pre-existing async fold/publish
  goroutine, which this task does not restructure). Fails without the `Put`
  line added at `processor.go:84`.

All three ran and passed (`go test ./movement/... -run
'TestTeleportCharacter|TestForCharacter' -v`) — output confirmed directly,
not taken from the implementer's report.

**Gap, non-blocking:** none of the three tests assert the *contents* of the
Kafka command (`CommandProducer`'s output — specifically that `Fh` is
published as `0`). The registry-write assertions are a reasonable proxy for
`x`/`y` (same values feed both `Put` and `CommandProducer`), but nothing in
this package asserts the `Fh=0` argument reaches the wire, which is the
single most safety-critical value in this task (it is what makes task 7's
`Move` branch to `UpdatePosition`). This is not a deviation from the brief
(the brief's test table specifies exactly these three assertions and no
command-content test) nor from this package's existing convention (no
existing test in `movement/` asserts `CommandProducer` output either —
checked `fold_test.go`, `processor_test.go`, `displacement_test.go`,
`action_test.go`). Flagged as a testing gap in the surface, not a defect
introduced by this commit.

## 6. Repo conventions

- `gofmt`-clean (implementer report + spot check of the diff hunks: mock
  struct fields realigned correctly under gofmt rules).
- DOM/MESSAGING conventions: publish goes through the existing
  `producer.ProviderImpl(...)(Env...)(CommandProducer(...))` pattern already
  used by `ForCharacter`/`ForPet`/`ForMonster` — no new pattern introduced.
- No new domain type/alias/constant was defined that duplicates
  `libs/atlas-constants` — `position.Position`/`position.Key` are pre-existing
  (task 5, read-only per this task's brief).
- Test setup reuses existing package-level builders rather than defining new
  `*_testhelpers.go` files, per convention.

## Verification performed

```
$ go build ./movement/...
(no output)
$ go test ./movement/... -run 'TestTeleportCharacter|TestForCharacter' -v
--- PASS: TestTeleportCharacter_WritesLastPosition (0.00s)
--- PASS: TestTeleportCharacter_NoClientboundBroadcast (0.00s)
--- PASS: TestForCharacter_WritesLastPosition (0.00s)
PASS
```

Did not run `tools/verify.sh` or `tools/lint.sh` (running concurrently
elsewhere per instructions); did not modify the working tree.

## Not evaluable

- The inner-portal handler that will call `TeleportCharacter` does not exist
  yet (later task) — cannot evaluate its call-site correctness from this
  commit alone; out of this unit's scope by design.

## Verdict

No blocking defects found. The cross-service seam (`fh=0` →
`UpdatePosition`, preserving stored foothold) was traced end-to-end through
the actual consumer code and confirmed correct. The registry write ordering
in both `ForCharacter` and `TeleportCharacter` is correct and race-free with
respect to fold failure. Mock matches convention. Tests are honest (fail
without the change) but stop short of pinning the emitted command's `Fh`
value — a non-blocking coverage gap, not a defect, since it also matches this
package's pre-existing convention.
