# Review: Task 1 — `IsPotionLocked` predicate (task-280)

Range reviewed: `5a7f4c59a..1161f19d6`
Files: `services/atlas-consumables/atlas.com/consumables/character/buff/model.go`,
`services/atlas-consumables/atlas.com/consumables/character/buff/model_test.go`

## Scope

Diff is exactly the two files named in the brief. No unrelated files touched.
`docs/tasks/task-280-stop-potion-consume-lock/agent-ledger.tsv` is untracked in
the worktree but not part of this commit range — out of scope for this review.

## Requirement-by-requirement

1. **Function added immediately after `IsZombified`.** `model.go:80-96` (new
   `IsPotionLocked`), directly following `IsZombified` (ends `model.go:77`).
   Confirmed via diff: the change is a pure append, zero lines removed —
   `IsZombified` body (`model.go:63-77`) is byte-identical pre/post. PASS.

2. **Uses the constant, never the string literal.** `model.go`:
   `c.Type == charconst.TemporaryStatTypeStopPortion`. Repo-wide grep for
   `"STOP_PORTION"` inside `services/atlas-consumables/` returns zero hits.
   PASS.

3. **Magnitude never consulted (FR-3).** The predicate only tests
   `c.Type == charconst.TemporaryStatTypeStopPortion`; `c.Amount` (the WZ `x`
   value) is never read. PASS.

4. **`IsZombified` not rewritten or generalised.** Diff confirms no edit to
   `IsZombified`; no shared `HasStat` helper was introduced — `IsPotionLocked`
   is a standalone loop, deliberately duplicating the shape per the brief's
   explicit instruction not to generalise. PASS.

5. **Signature matches contract for Task 3.** `func IsPotionLocked(bs []Model) bool`
   — matches "Produces" interface in the brief exactly. PASS.

6. **All seven brief table rows present in full, not abbreviated.** Verified
   by diff inspection (`model_test.go`, new lines): `unexpired stop portion`,
   `expired stop portion`, `no-expiry stop portion`, `unexpired non-stop-portion`,
   `stop portion not first change`, `empty slice`,
   `expired stop portion alongside unexpired speed` — seven `name:` entries,
   each with the exact buff construction and `want` value tabled in the brief.
   No ellipsis, no fewer/more cases. PASS.

7. **Test-then-implementation (TDD) honesty.** Report's Step 2 evidence shows
   the pre-implementation run failing with
   `character/buff/model_test.go:152:14: undefined: IsPotionLocked` — a build
   failure that only resolves once `IsPotionLocked` exists, so the test cannot
   pass against the pre-change tree. Re-ran locally in the current worktree
   (post-implementation) — all 7 subtests plus the untouched `TestIsZombified`
   (7 subtests) and `TestExpiredHonoursNoExpiry` pass:
   ```
   ok  	atlas-consumables/character/buff	0.060s
   ```
   PASS.

8. **No collateral damage to existing tests.** `TestIsZombified` and
   `TestExpiredHonoursNoExpiry` both pass unchanged. PASS.

9. **Formatting / vet cleanliness.** `gofmt -l` on both changed files:
   no output (clean). `go vet ./character/buff/...`: clean. PASS.

## Correctness of the predicate itself

- Skips expired buffs via `b.Expired()` before inspecting changes — correct,
  matches `IsZombified` shape and the "no-expiry" semantics already pinned by
  `TestExpiredHonoursNoExpiry`.
- Inner loop scans all `b.changes`, not just index 0 — correctly handles the
  "stop portion not first change" case.
- `nil` slice input: range over nil is a no-op in Go, returns `false` —
  matches the "empty slice" test row.
- Multi-buff slice (expired STOP_PORTION + unexpired SPEED): loop continues
  past the expired buff, evaluates the SPEED buff's changes, finds no
  STOP_PORTION, returns `false` — matches the tabled expectation.

No boundary or nil-handling defects found within this predicate's surface.

## Not evaluable

- Task 3 (the actual consume-gate caller of `IsPotionLocked`) is out of scope
  for this review — it is a separate task unit per the plan and not present
  in this diff. Whether `IsPotionLocked` is wired correctly into the consume
  path cannot be assessed from this range.

## Verdict

APPROVED — every brief requirement is met with cited evidence, all seven
table rows are present verbatim, the constant is used with no string-literal
leak, `IsZombified` is untouched, and local test/vet/fmt runs confirm the
work as committed.
