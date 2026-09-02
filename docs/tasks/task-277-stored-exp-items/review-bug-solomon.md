# Review: bug-solomon-no-effect-no-feedback, `## Fix` sections 1 and 2

Range reviewed: `0aeb0f33a..993b6af7b` (`1e33a1a78`, `993b6af7b`).
Scope: `## Fix` sections 1 (TODO.md re-ingest follow-up) and 2 (per-rejection
player feedback). Section 3 is operational and explicitly out of scope.

## Summary

Both commits do what the brief specifies. The cross-service string seam was
traced by hand end to end and the three literals match byte-for-byte on both
sides, each pinned by a dedicated test in both modules. The ordering contract
(all three rejections before `ConsumeItem`) holds. The pre-existing
`CONSUME_FAILED -> actionUnstick` default is untouched. Both modules build and
their relevant test packages pass.

## Fix 1 — TODO.md re-ingest follow-up + prd.md tick

- `docs/tasks/task-277-stored-exp-items/prd.md:331` — checkbox flipped from
  `[ ]` to `[x]` for "If a WZ `spec` field was added to `atlas-data`, a tenant
  re-ingest follow-up is recorded in `docs/TODO.md`." — PASS.
- `docs/TODO.md` — the `## task-277 follow-up` section (around line 681) was
  reworded/expanded with two unchecked action items (re-ingest, then verify
  per tenant) — PASS, content-wise it satisfies the acceptance item.

**Finding (non-blocking, informational):** the bug doc's own root-cause
narrative (`bug-solomon-no-effect-no-feedback.md:86-90`) states "no such entry
exists in `docs/TODO.md`," but `git log` shows the `## task-277 follow-up`
section already existed on this branch before the reviewed range, added by
`49d3a05ca` ("docs: record stored-EXP implementation and the tenant re-ingest
follow-up"), an ancestor of `0aeb0f33a`. The fix commit reworded an existing
entry rather than creating a new one from scratch. This is a factual
inaccuracy in the bug report's diagnosis, not a defect in the fix — the
resulting TODO.md content and the prd.md tick both satisfy the acceptance
item's actual requirement (an entry exists, recorded, unchecked action items
present). Flagging so the bug doc's Cause narrative isn't taken as gospel
elsewhere.

## Fix 2 — per-rejection player feedback

### Producer side (`atlas-consumables`)

- `consumable/solomon.go:29-40` — three sentinel errors declared
  (`ErrSolomonNoExperience`, `ErrSolomonLevelExceeded`,
  `ErrSolomonBalanceNotEmpty`), replacing the three `errors.New(...)`
  literals in place (`solomon.go:80,85,90`) — PASS.
- **Ordering contract** — confirmed by hand, `solomon.go:78-94`: the
  spec/exp check (78-80), level-gate check (83-85), and balance check
  (88-90) all `return d.onError(...)` strictly before
  `d.compartment.ConsumeItem(...)` at line 92. No Writ can be destroyed on
  any of the three rejection paths — PASS.
- `kafka/message/consumable/kafka.go:132-141` — three producer-side
  constants added: `ErrorTypeSolomonNoExperience = "SOLOMON_NO_EXPERIENCE"`,
  `ErrorTypeSolomonLevelExceeded = "SOLOMON_LEVEL_EXCEEDED"`,
  `ErrorTypeSolomonBalanceNotEmpty = "SOLOMON_BALANCE_NOT_EMPTY"` — PASS.
- `consumable/processor.go:509-521` (`consumeErrorType`) — three
  `errors.Is` arms added ahead of the `ErrorTypeConsumeFailed` fallthrough,
  matching each sentinel to its constant — PASS. Confirmed the fallthrough
  (`return consumable.ErrorTypeConsumeFailed`) is otherwise unchanged — PASS.
- **Reach confirmed**: `ConsumeError` (`processor.go:486-499`) is the sole
  production caller path — `solomon.go:114` binds `onError` to
  `ConsumeError` with no wrapping — and it calls
  `consumeErrorType(err)` directly on the raw error returned by
  `d.onError(ErrSolomonXxx)`. `errors.Is` therefore matches the un-wrapped
  sentinel exactly; no wrapping/unwrapping gap — PASS.

### Consumer side (`atlas-channel`)

- `kafka/message/consumable/kafka.go:102-111` — the same three constants
  declared independently, with a comment explicitly flagging the
  hand-mirror risk — PASS, and **literal-for-literal identical** to the
  producer's strings (verified via diff, both read
  `"SOLOMON_NO_EXPERIENCE"`, `"SOLOMON_LEVEL_EXCEEDED"`,
  `"SOLOMON_BALANCE_NOT_EMPTY"`).
- `kafka/consumer/consumable/consumer.go:107-146` — new `actionSolomonRejected`
  enum value, three message constants (`SolomonNoExperienceMessage`,
  `SolomonLevelExceededMessage`, `SolomonBalanceNotEmptyMessage`), and
  `solomonRejectionMessage(errorType string) string` resolving each — PASS.
- `consumer.go:159-171` (`consumableErrorAction`) — all three new constants
  route to `actionSolomonRejected`; the `default: return actionUnstick`
  branch, and the explicit `ErrorTypePotionLocked -> actionUnstick` arm,
  are both unchanged — PASS. `CONSUME_FAILED -> actionUnstick` default is
  preserved (no case added for it, still falls to default) — PASS, matches
  the brief's explicit "do not change" instruction.
  (`bug-solomon-no-effect-no-feedback.md:167-168`)
- `consumer.go:199-206` — new `case actionSolomonRejected:` arm announces
  `charcb.CharacterStatusMessageWriter` +
  `charpkt.CharacterStatusMessageOperationSystemMessageBody(solomonRejectionMessage(e.Body.Error))`,
  then calls `unstick(s)` — matches the Water-of-Life precedent shape
  exactly (`socket/handler/water_of_life.go:57`,
  `kafka/consumer/pet/consumer.go:561`) — PASS.

### Test honesty — both sides pin the literals

- Producer: `kafka/message/consumable/kafka_test.go` (new file,
  `TestSolomonErrorWireValues`) asserts each constant equals the literal
  wire string — PASS, this is a real pin (a typo in the producer constant
  would fail this test even with nothing on the consumer side to compare
  against).
- Consumer: `kafka/consumer/consumable/consumer_test.go` adds
  `TestSolomonErrorWireValues` (same three literal assertions) and
  `TestSolomonRejectionMessage` (message-per-type, plus the unrecognized-type
  empty-string fallback) — PASS. Also extends the existing
  `TestConsumableErrorAction` table with the three new
  type→`actionSolomonRejected` rows — PASS, and the untouched
  `CONSUME_FAILED -> actionUnstick` / `"" -> actionUnstick` /
  `"SOMETHING_ELSE" -> actionUnstick` rows remain in the table
  (`consumer_test.go:25-30` region) — confirms the default path is pinned,
  not just added-to.
- `consumable/solomon_test.go` — new
  `TestConsumeSolomonRejectionErrorTypes` drives `consumeSolomon` through
  all three rejection paths via table cases (`spec/exp absent`, `level
  exceeds maxLevel`, `balance already non-zero`), asserts
  `consumeErrorType(err)` returns the matching new constant, AND asserts
  `ConsumeItem`/`CreditStoredExperience` were never called — this is a real
  test of both the classification AND the ordering contract together, not
  just coverage — PASS.
- **This seam has no shared constant** (confirmed: `grep` for
  `SolomonNoExperience`/`SOLOMON_NO_EXPERIENCE` finds two independent
  declaration sites, exactly as the brief anticipated and the code comments
  on both sides acknowledge) — the two `TestSolomonErrorWireValues` tests
  are the only thing pinning the two sides in agreement. Both exist and
  both pass. This is a residual maintenance risk inherent to the design
  choice made in the brief (no shared library constant), not a defect in
  this implementation — noted, not blocking.

### Build/test verification (module-local, not a substitute for `verify.sh`)

- `atlas-consumables`: `go build ./...` clean;
  `go test ./consumable/... ./kafka/...` — all packages `ok` or "no test
  files", including `kafka/message/consumable` (the new
  `TestSolomonErrorWireValues`).
- `atlas-channel`: `go build ./...` clean; `go test
  ./kafka/consumer/consumable/...` — `ok`.

### Commit 993b6af7b — level-gate message correction

`consumeSolomon` rejects when `c.Level() > ci.MaxLevel()` (too high, not too
low). The first commit's message read "You are not experienced enough to use
the Writ of Solomon." — inverted. `993b6af7b` corrects it to "Your level is
too high to use the Writ of Solomon." at
`kafka/consumer/consumable/consumer.go:133`. Confirmed via `git show` diff —
PASS, and correctly matches the actual gate direction in
`solomon.go:83-85`.

## Not evaluable

- **Live end-to-end verification** (does a real Writ of Solomon, once
  re-ingested, actually bank EXP and does the message actually render
  client-side) — explicitly out of scope per Fix section 3 / the bug doc's
  own "Resolution" note ("Live re-test: NOT done, and cannot be done until
  the re-ingest runs"). Not evaluable from this diff alone; depends on the
  operational re-ingest step.
- **Message copy approval.** The brief flags the three strings as needing
  the user's sign-off ("Message copy still needs the user's approval"). Not
  a code-correctness question; noted as still open per the bug doc itself,
  not a finding against this diff.

## Verdict rationale

No blocking defects found. The cross-service seam was traced by hand per the
dispatch brief: literals match byte-for-byte, both sides pin them in tests,
and `consumeErrorType`'s classification reaches `consumableErrorAction`'s new
arm for exactly the three intended errors, with the `CONSUME_FAILED ->
actionUnstick` default unchanged. The ordering contract (reject before
`ConsumeItem`) holds and is itself asserted by a new test. One non-blocking,
informational note about a factual inaccuracy in the bug doc's own root-cause
narrative (Fix 1) does not affect the correctness of the change.
