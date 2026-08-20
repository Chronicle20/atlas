# Review: Task 2 — Legacy tenant reconciliation

Range reviewed: `4074a7f7a..15910ad9e` (single commit `15910ad9e`).
Brief: `.superpowers/sdd/plan/task-2-brief.md`
Report: `.superpowers/sdd/plan/task-2-report.md`

## Scope

`git diff --stat 4074a7f7a..15910ad9e`:

```
libs/atlas-env/tenants.go      | 10 ++++++++++
libs/atlas-env/tenants_test.go | 34 ++++++++++++++++++++++++++++++++++
2 files changed, 44 insertions(+)
```

Matches the brief's declared file set exactly. No other files touched.
`scope_confirmed`: reviewed the full diff (both files, all four hunks) plus
`libs/atlas-env/registry.go` (to confirm `ApplyTenant` is untouched, D4) and
the two live external callers of `env.Reconcile`
(`libs/atlas-rest/server/handler.go:151`,
`libs/atlas-kafka/consumer/header.go:93`) and their test suites, to check the
cross-service seam per the task instructions.

## Findings

### 1. PASS — D4 respected, `ApplyTenant` untouched

`git diff 4074a7f7a..HEAD -- libs/atlas-env/registry.go` is empty. The new
arm lives entirely inside `Reconcile` in `tenants.go:67-75`, exactly at the
brief's specified insertion point (immediately after the `!known` early
return, immediately before `if headerEnv == ""`).

### 2. PASS — implementation matches the brief verbatim

`libs/atlas-env/tenants.go:67-75`:

```go
if tenantEnv == "" {
    return headerEnv, nil
}
```

placed before the pre-existing `if headerEnv == "" { return tenantEnv, nil }`
and `if headerEnv != tenantEnv { ... }` arms, unchanged. Comment matches the
brief's text.

### 3. PASS — FR-3.2 holds by construction

Confirmed by reading the control flow: the `headerEnv != tenantEnv` mismatch
arm at `tenants.go:78-81` is reached only when `tenantEnv != ""` (guarded by
the new arm) and `headerEnv != ""` (guarded by the pre-existing arm above
it). `TestReconcileStillRejectsTwoNonEmptyDisagreements` exercises this with
two distinct non-empty values (`pr-123` vs `pr-1411`) and asserts
`errors.Is(err, ErrEnvironmentMismatch)`.

### 4. PASS — test honesty, verified by reverting the arm

Temporarily removed the new `if tenantEnv == "" { return headerEnv, nil }`
block and re-ran `go test ./... -run TestReconcile -v`:

```
--- FAIL: TestReconcileTrustsTheHeaderForALegacyTenant (0.00s)
    tenants_test.go:73: got ("", environment header disagrees with the
    tenant's environment: header="pr-1411" tenant=""(t-1)), want
    ("pr-1411", nil)
```

All other tests (including the two other new cases) passed against the old
code, exactly as the report claims — those two exercise pre-existing arms
and only the FR-3.1 case is a genuine RED→GREEN test. File was restored
after the check (`git diff --stat tenants.go` empty afterward).

### 5. PASS — pre-existing `Reconcile` tests unmodified

The four pre-existing tests (`TestReconcileAgreesWhenHeaderMatchesTenant`,
`TestReconcileRejectsADisagreement`,
`TestReconcileDerivesFromTenantWhenNoHeaderIsPresent`,
`TestReconcileWithAnUnknownTenantTrustsTheHeader`,
`TestReconcileWithNeitherIsTheLegacyValue` — five, brief said "four", minor
miscount in the brief/report but immaterial) appear only in the diff's
context lines, not touched. All still pass (confirmed via targeted run
above). No test was loosened to accommodate the new arm.

### 6. PASS — no outcome change for non-legacy records

Traced all four reachable branches after the new arm:
- `tenantEnv == "" ` → new arm, returns `headerEnv` (this is the only newly
  changed outcome — previously fell through to compare against `""` and
  either matched header=="" or hard-mismatched any non-empty header).
- `tenantEnv != "" && headerEnv == ""` → unchanged, returns `tenantEnv`.
- `tenantEnv != "" && headerEnv != "" && headerEnv == tenantEnv` → unchanged,
  returns `headerEnv`.
- `tenantEnv != "" && headerEnv != "" && headerEnv != tenantEnv` → unchanged,
  hard mismatch error.

Only the legacy (`tenantEnv == ""`, `known == true`) case changes behavior,
which is exactly FR-3.1's intent.

### 7. PASS — cross-service seam check (D4 / consumer contracts)

`env.Reconcile` has exactly two live callers in the tree:
`libs/atlas-rest/server/handler.go:151` and
`libs/atlas-kafka/consumer/header.go:93`. Searched both packages' test
suites (`libs/atlas-rest/server/handler_test.go`,
`libs/atlas-kafka/consumer/header_env_test.go`,
`libs/atlas-kafka/consumer/gate_test.go`) for `ApplyTenant` calls using an
empty environment (the legacy case this task changes) — none found; all use
`env.Id("pr-123")` or a non-empty tenant environment. No consumer test pins
the old legacy-tenant-mismatch behavior, so this change does not silently
break an existing assertion elsewhere in the tree. (Whether those consumers
need new legacy-tenant test coverage of their own is presumably a later
task's concern — out of this task's declared scope of
`libs/atlas-env/tenants.go` + `tenants_test.go`.)

### 8. PASS — backend guideline conventions

Test setup uses the existing `NewMapRegistry(...)` + `ApplyTenant(...)`
pattern already established by the four pre-existing tests in the same file
— no new `*_testhelpers.go`, no test-only constructor introduced. `Reconcile`
remains a pure function; no new mutable/shared state added.

## Not evaluable

None — the full unit (one function, one file's test additions) was within
reach of direct inspection and a targeted test run.

## Verdict

APPROVED. Diff is minimal, matches the brief exactly, D4 is respected, the
new test is a genuine RED→GREEN case (verified by reverting the arm), the
three pre-existing behaviors (agreement, hard mismatch, no-header-derives)
are provably unaffected by control-flow tracing, and no downstream consumer
test was loosened to accommodate the change.
