# Task 14 batch D review — `services/atlas-tenants/atlas.com/tenants/configuration`

Range reviewed: `f1ef34e..4f6252b` (`c4ff02806` test commit, `4f6252b28` docs commit).
Brief: `.superpowers/sdd/plan/task-14-brief-d.md`. Report: `.superpowers/sdd/plan/task-14-report-d.md`.

## Scope

`git diff --stat f1ef34e..4f6252b`:

```
docs/tasks/task-263-backend-guideline-conformance/handwork-notes.md              |   4 +
services/atlas-tenants/atlas.com/tenants/configuration/rest_test.go              | 317 +++++++++++++++++++++
2 files changed, 321 insertions(+)
```

Matches the brief exactly: `rest_test.go` (new tests only, no `rest.go` touch) plus the
handwork-notes entry. No `Transform*` was added — confirmed by the diff stat showing zero
changes to `rest.go`.

## Findings

### 1. `rest.go` unchanged — PASS

`git diff --exit-code f1ef34e..4f6252b -- services/atlas-tenants/atlas.com/tenants/configuration/rest.go`
returned empty (no output, exit 0). Read-only constraint honored.

### 2. Non-tautology — PASS, independently verified for 3 of 5 new tests

I ran my own live mutations (not the implementer's — different fields, to get independent
evidence) against a working copy of `rest.go`, restoring the original via `git diff --exit-code`
after each:

- **Route**: dropped the `attributes["startMapId"]` read in `TransformRoute` (hardcoded
  `startMapId := uint32(0)`, no `if val, ok := ...` block). `go test -run
  TestRouteTransformExtractRoundTrip` failed: `rest_test.go:337: startMapId = 0, want 101000000`.
- **MtsConfig**: dropped the `attributes["commissionRate"]` read in `TransformMtsConfig`. Failed:
  `rest_test.go:465: commissionRate = 0, want 0.05`.
- **InstanceRoute**: dropped the `attributes["capacity"]` read in `TransformInstanceRoute`.
  Failed: `rest_test.go:582: capacity = 0, want 6`.

After all three mutate/test/revert cycles, `git diff --exit-code
services/atlas-tenants/atlas.com/tenants/configuration/rest.go` was clean — the working tree
matches the committed `rest.go` byte-for-byte. These three plus the implementer's own reported
mutations for Vessel (`routeBID`) and ImprintConfig (`pendingExpiryHours`) give field-level
failure evidence for all 5 new tests. None is tautological.

### 3. No duplicate tests — PASS

- `trade_config_test.go:48` `TestTradeConfigRoundTrip` — read directly; it calls
  `ExtractTradeConfig`, round-trips the result through real `encoding/json` (`jsonRoundTrip`), then
  `TransformTradeConfig`, and asserts `Id`, `TaxEnabled`, `MaxStagedItems`, `MinTradeLevel`, etc.
  This is genuine Extract→Transform pair coverage; the implementer's claim to leave `TradeConfig`
  alone is correct and no new test duplicates it.
- `rankings_test.go:37` `TestRankingsCreateGetRoundTrip` — read directly; it calls
  `p.CreateRankings` (which invokes `TransformRankings`) then `p.GetRankings` (which invokes
  `ExtractRankings`) and asserts `recomputeIntervalMinutes` survives the cycle. It is an indirect,
  DB-mediated round trip but genuinely exercises both halves of the pair. Correctly counted as
  covered; no duplicate written.
- `KiteConfig` (`rest_test.go:252`) and `RpsReward` (`rest_test.go:14`) were left untouched, as
  instructed.

### 4. `map[string]interface{}` traps — PASS

- Fixtures for all 5 new tests populate every mapped key of their respective attribute sets with
  distinct, non-zero values (verified by reading each fixture against the corresponding `Extract*`
  body in `rest.go`, e.g. `TestRouteTransformExtractRoundTrip` covers all 9 `RouteRestModel`
  attribute fields, `TestInstanceRouteTransformExtractRoundTrip` covers all 10).
- Concrete numeric types match what `Transform*` actually reads: fixtures use `float64(...)` for
  every attribute `Transform*` type-asserts as `float64` (confirmed by reading `TransformRoute`,
  `TransformMtsConfig`, `TransformInstanceRoute`, `TransformImprintConfig` bodies directly), and
  assertions on the `Extract*` side use `uint32`/`int`/`float64` matching what each `Extract*`
  literally emits (e.g. `attrs["maxActiveListings"] != 20` against an untyped constant compared to
  an `int` field — correct; `attrs["listingFee"] != uint32(1000)` against a `uint32` field —
  correct).
- Tenant-derived uuid for `Route`, `Vessel`, `InstanceRoute` is asserted separately
  (`rm.Uuid != wantUuid` via `tenant.DerivedId(...)`) before the map-equality checks, not folded
  into a weakened map comparison — matches the brief's requirement exactly
  (`rest_test.go:313-316`, `:387-390`, `:548-551`).

### 5. Exemption note — PASS

`docs/tasks/task-263-backend-guideline-conformance/handwork-notes.md:26` (Batch D entry):
repo-relative paths only, matches the batch-A/B/C form (package, no-RestModel statement, wire
type list, `Extract*` line numbers, provided-`Transform*` statement, coverage-completed-by list,
domain-side note, out-of-scope wire types). Correctly states all 9 `Transform*` pre-existed and no
new `Transform` was written. `TradeTaxTierRestModel`/`RpsRewardRungRestModel` correctly named as
inline-mapped, out of scope for a standalone `Transform` — consistent with batch-B's pattern for
sub-model wire types.

### 6. `processor_test.go:848` flagged observation — CONFIRMED, accurate

Read `processor_test.go:848-875` (`TestMtsConfigRoundTrip`) directly. It calls
`processor.createMtsConfig`, `processor.getMtsConfigById`, and `configuration.TransformMtsConfig`
— it never calls `ExtractMtsConfig` anywhere in the function body. The implementer's claim that
this pre-existing test is not genuine Extract/Transform pair evidence is correct, and does not
undercut this batch's new `TestMtsConfigTransformExtractRoundTrip`, which does exercise both
halves directly.

### Build/format/test gate — PASS

Independently re-ran (not just trusting the report):

```
$ go test ./configuration/... -run RoundTrip -v   → all 12 round-trip tests PASS
$ tools/lint.sh --check --fmt --go services/atlas-tenants/atlas.com/tenants   → lint.sh: OK
$ go build ./... && go vet ./... && go test ./...   → all packages ok, no failures
```

## Not evaluable

None — the batch's surface (5 new tests, 1 doc entry, zero implementation) was fully within
reach for direct verification.

## Notes (non-blocking)

- Working tree at review time had unrelated pre-existing local modifications
  (`docs/tasks/task-263-backend-guideline-conformance/agent-ledger.tsv`,
  `progress.md`, and an untracked `task-14c-review.md`) that are outside the
  `f1ef34e..4f6252b` range and not part of this unit — noted for completeness, not a finding
  against this batch.

## Verdict

APPROVED. All 5 findings categories pass with direct evidence; independent mutation testing on 3
of the 5 new tests confirms non-tautology, matching the bar batch B set. `rest.go` is confirmed
byte-identical before and after review. Exemption note is well-formed and accurate. The flagged
`processor_test.go:848` observation is independently confirmed true.
