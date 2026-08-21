# Review: Task 1 — `atlas-data` absent-`selfDestruction` sentinel

Commit range: `2ed03e84f..1adaf6c9c` (single commit `1adaf6c9c`)
Brief: `.superpowers/sdd/plan/task-1-brief.md`
Report: `.superpowers/sdd/plan/task-1-report.md`

## Scope confirmed

The diff touches exactly the two files named in the brief:

- `services/atlas-data/atlas.com/data/monster/reader.go` (+6/-1)
- `services/atlas-data/atlas.com/data/monster/reader_test.go` (+2/-2)

`rest.go` is untouched, matching the brief's read-only instruction. No other file in the
commit range. Scope matches the brief exactly — no drift.

## Spec compliance (against `task-1-brief.md`)

1. **Sentinel value and field order** — PASS.
   `reader.go:214`: `return selfDestruction{Action: 0, RemoveAfter: -1, Hp: -1}` for the
   absent-node branch, replacing the prior `selfDestruction{}` zero value. Struct field
   order confirmed at `rest.go:91-95`: `Action byte`, `RemoveAfter int32`, `Hp int32` — matches
   the required `{Action, RemoveAfter, Hp}` order, and the literal is keyed (`Action:`,
   `RemoveAfter:`, `Hp:`) rather than positional, so field-order drift in the struct
   definition would not silently miscompile the semantics.

2. **Test expectation updated** — PASS.
   `reader_test.go:1267-1268`: assertion changed from `selfDestruction{0, 0, 0}` to
   `selfDestruction{0, -1, -1}`, matching the brief's Step 1 instruction verbatim.

3. **TDD evidence (RED then GREEN)** — PASS, and independently reproduced.
   Re-ran `go test ./monster/ -run TestRead -v` and `go build ./... && go test ./monster/`
   in the current worktree state (post-commit): both green.
   ```
   --- PASS: TestReader (0.00s)
   ok  	atlas-data/monster	0.036s
   ```
   The report's claimed RED output (`got {Action:0 RemoveAfter:0 Hp:0}, expected
   {Action:0 RemoveAfter:-1 Hp:-1}`) is consistent with reverting only the `reader.go`
   hunk while keeping the updated test — plausible and matches the brief's Step 2
   expected output exactly. Not independently re-run (would require reverting the
   committed fix), but the mechanism is straightforward and low-risk to trust here.

4. **Test honesty** — PASS.
   Confirmed the `TestReader` fixture (`testXML` at `reader_test.go:15`) contains no
   `selfDestruction` node — the only occurrences of the string `selfDestruction` in
   `reader_test.go` are the two assertion lines, so `getSelfDestruction`'s absent-node
   branch is genuinely exercised. Before the fix, the old test asserted `{0,0,0}` which
   the old zero-value return also produced, so the *old* test was not discriminating
   between "absent" and "present with hp 0" — the new test now is, since `-1` sentinels
   are produced only by the absent-node branch (the present-node branch defaults `hp`/
   `removeAfter` to `-1` only when the XML field itself is omitted, which is a different,
   legitimate code path per the brief's own comment).

5. **`rest.go` unchanged** — PASS. No diff to `rest.go`; grep confirms it is not part of
   the commit.

6. **No stubs / TODOs / invented values** — PASS. The added comment cites the design
   doc (`task-253 design D2`) rather than an unverifiable IDA address, which is
   appropriate here since this comment describes the *design decision*, not a decoded
   game-data field meaning — the "cite an IDA address" constraint applies to comments
   asserting a dead-type's game meaning, not to comments citing this repo's own design
   doc.

7. **Presence test `Hp > -1 || RemoveAfter > -1`** — Not applicable to this task. No
   consumer of `selfDestruction` presence exists in the diff or in this module (grep
   across `monster/*.go` shows `SelfDestruction`/`selfDestruction` referenced only in
   `rest.go` (struct/field decl) and `reader.go` (producer)). This constraint binds a
   downstream task that reads presence; it is out of scope for Task 1 and is recorded
   under Not evaluable, not silently passed.

## Task quality

- Small, single-purpose, keyed struct literal (avoids positional-field-order footguns).
- Comment is precise about *why* (collision between "absent" and "hp:0 present"), cites
  the design decision, and does not overreach into claims about downstream consumers.
- Commit contains only the two named files (verified via `git diff --stat` on the range).
- No line-ending changes: both the base and the fixed function body use `\n` (LF) with
  tab indentation; the change doesn't introduce CRLF or normalize anything.
- No `*_testhelpers.go` added; the existing test file's inline literal-style assertions
  are preserved rather than restructured.
- Immutable-domain-model constraint: `selfDestruction` is a value type (not a pointer),
  constructed once and returned — consistent with the rest of `reader.go`'s sibling
  functions (e.g. `getFirstAttack`, `getBanish` pattern nearby).

## Not evaluable

- The `Hp > -1 || RemoveAfter > -1` presence-test convention (design §2.6) has no
  consumer in this task's diff to check it against; deferred to whichever later task
  in the plan reads `selfDestruction` presence (e.g. a channel/spawn-logic task).
- The RED-state test output in the report was not independently re-triggered (would
  require locally reverting the committed fix); accepted on the strength of the
  reasoning above rather than direct re-execution.

## Verdict

Both spec compliance and task quality: no blocking findings.
