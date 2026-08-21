# Review: Task 6 — Populate the expression command's `TransactionId`

**Commit range:** `a9f733017..9a31743` (single commit `9a31743a0`)
**Brief:** `.superpowers/sdd/plan/task-6-brief.md`
**Report:** `.superpowers/sdd/plan/task-6-report.md`

## Scope confirmed

Diff touches exactly the three files listed in the brief's Files section:

- `services/atlas-channel/atlas.com/channel/kafka/message/expression/kafka.go`
- `services/atlas-channel/atlas.com/channel/character/expression/producer.go`
- `services/atlas-channel/atlas.com/channel/character/expression/producer_test.go`

`services/atlas-expressions/.../kafka.go` was correctly left untouched (read-only per brief). No wire-format files, no `character_cash_item_use.go`, no `atlas-expressions` `model.go`/`registry.go` changes. `SetCommandProvider`'s signature (`characterId, f, expression, duration, byItemOption`) established by Task 4 is unchanged — confirmed at `producer.go:15`.

## Findings

### PASS — `TransactionId` field added correctly, name/tag/order match consumer

`services/atlas-channel/.../kafka/message/expression/kafka.go` `Command` struct now has `TransactionId uuid.UUID \`json:"transactionId"\`` as the first field. Compared directly against `services/atlas-expressions/atlas.com/expressions/kafka/message/expression/kafka.go` (`Command` struct, lines 30-40): field name, JSON tag, and type (`uuid.UUID`) match exactly, and field ordering mirrors the consumer's struct (matches brief's explicit instruction to mirror atlas-expressions' field order). JSON is tag-driven so struct field order doesn't affect wire compatibility either way, but the mirroring is a legitimate readability convention and was followed.

### PASS — producer sets a fresh UUID per command, not zero/shared

`character/expression/producer.go:19` — `TransactionId: uuid.New()` is set inside the `Command` literal built fresh on every call to `SetCommandProvider`. `uuid.New()` is evaluated at struct-literal construction time inside the function body, not hoisted to a package-level var or computed once and captured by a closure, so each invocation of `SetCommandProvider(...)` yields a distinct value. Verified empirically: two successive calls with identical arguments (`SetCommandProvider(1000, f, 8, -1, false)()` called twice) produced two different `TransactionId` values in the test run.

Grepped the module for other producers constructing `expression2.Command{...}` literals (`grep -rn "Command{" --include='*.go'`) — only one non-test call site exists (`character/expression/producer.go:19`), so there is no other unset code path left carrying the zero UUID.

### PASS — test asserts something real, not tautological

`TestSetCommandProviderSetsTransactionId` (`producer_test.go`) calls the real production function, marshals through `producer.SingleMessageProvider` and unmarshals the JSON bytes back into `expression2.Command` (round-tripping through the actual wire encoding, not asserting against an in-memory struct field the test itself set). It asserts:
- `cmd1.TransactionId != uuid.Nil`
- `cmd2.TransactionId != uuid.Nil`
- `cmd1.TransactionId != cmd2.TransactionId`

The third assertion is the one that actually proves "fresh UUID per call" rather than "same fixed non-nil UUID reused" — a bug class a naive "just check non-nil" test would miss. Confirmed by direct reasoning: reverting the `TransactionId: uuid.New()` line (leaving the field undeclared, matching pre-change state) would fail to compile per the brief's Step 2 expectation, and even a hypothetical package-level `var txId = uuid.New()` regression would be caught by the distinctness assertion. Ran the test directly:

```
$ go test ./character/expression/... -run TestSetCommandProviderSetsTransactionId -v
=== RUN   TestSetCommandProviderSetsTransactionId
--- PASS: TestSetCommandProviderSetsTransactionId (0.00s)
PASS
```

### PASS — build and existing test suite unaffected

`go build ./...` succeeds in the module root. No other file in `atlas-channel` builds an `expression2.Command` literal, so no other call site was left needing (or missing) the new field.

### Note (non-blocking) — RED step not independently executed

The report's TDD Evidence section states the implementer did not run a standalone RED (compile-failure) step before adding the field, reasoning that the three files were "part of the same brief-specified diff." This is a minor process deviation from the brief's literal Step 2/Step 3 split, but it's low-risk here: the field addition is a single, unambiguous one-line struct change, and the final GREEN run (both quoted in the report and independently reproduced above) proves the test exercises the new field. Not a correctness defect.

## Not evaluable

None — the full surface of the brief (field addition, producer set, and test) was reviewed directly against source and re-run.

## Verdict

APPROVED. Small, correctly scoped commit: fresh per-call UUID, field name/tag/order matches the atlas-expressions consumer contract, `SetCommandProvider` signature untouched, and the new test's distinctness assertion is a real, non-tautological check that would catch both "never set" and "same value reused" regressions.
