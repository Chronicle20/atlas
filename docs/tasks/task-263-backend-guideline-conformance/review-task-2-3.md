# Review: Task 2 (`classify` subcommand) and Task 3 (merge alias constructors)

Range reviewed: `1b1631c8a..8c1ce0804`
Commits: `06e1a9120` (chore: derive conformance work classification), `11960df7a` (fix: persist
classify ledgers and add classifier tests), `8c1ce0804` (refactor: merge NewModelBuilder
aliases into NewBuilder).

Scope confirmed: matches the plan. `06e1a9120`/`11960df7a` implement plan.md "## Task 2" (the
`classify` subcommand, its tests, and the three derived TSVs + ledgers). `8c1ce0804` implements
"## Task 3" (merging the six `NewModelBuilder`/`NewBuilder` alias pairs in `atlas-channel`). No
extra, out-of-brief changes found in the diff.

The count-divergence between the derived TSVs and the design's §2 prose estimates was already
adjudicated by the controller in `progress.md` ("the derived TSVs are authoritative going
forward"); this review does not revisit that adjudication and instead checks the classifier's
rules against the plan's rule tables.

## Task 3 — merge the six alias constructor pairs

**PASS.** All six packages merged correctly:

- `channel/builder.go:23-31`, `note/builder.go:16-24`, `world/builder.go:17-21` — the
  parameter-less pairs are collapsed into a single `NewBuilder` carrying the original body and
  a plain `// NewBuilder creates a new builder instance` comment; the aliasing `NewBuilder` and
  the old `NewModelBuilder` are both gone. Confirmed via `git show 8c1ce0804` on each file.
- `compartment/builder.go:19-30`, `inventory/builder.go:15-27`,
  `transport/route/builder.go:27-42` — the parameterized pairs (identical signatures) merged
  the same way; `inventory/builder.go`'s result matches the plan's literal expected snippet
  exactly, including the untouched `BuilderSupplier`.
- No stray `NewModelBuilder` remains in any of the six packages:
  `grep -rn --include='*.go' 'NewModelBuilder' <six package dirs>` → no output.
- Call sites repointed, including cross-package callers the plan called out by name:
  `services/atlas-channel/atlas.com/channel/respawn/plan_test.go` (4 call sites: `compartment.NewBuilder`,
  `channelInventory.NewBuilder`) and
  `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_check_name_change_possible_test.go:154`
  (`channelworld.NewBuilder`). `git diff --stat 1b1631c8a..8c1ce0804 -- services/atlas-channel`
  shows exactly the 14 files the plan's "Files" section and Step 3 imply — no unrelated files
  touched.
- Repo-wide grep for `channel.NewModelBuilder(`/`note.NewModelBuilder(`/`world.NewModelBuilder(`/
  `compartment.NewModelBuilder(`/`inventory.NewModelBuilder(`/`route.NewModelBuilder(` across
  `services` and `libs` still returns hits, but every one resolves to a **different** package
  that happens to share a base name and was never in Task 3's scope:
  - `services/atlas-channel/atlas.com/channel/cashshop/inventory` and
    `.../cashshop/inventory/compartment` are distinct packages (own `package inventory` /
    `package compartment`, own `modelBuilder`, own field sets) that only ever declared
    `NewModelBuilder` — they never had the "alias for backward compatibility" `NewBuilder`, so
    they are not part of the six pairs the plan named.
  - `services/atlas-world/atlas.com/world/{world,channel}` is a wholly separate Go module
    (different service) with its own `NewModelBuilder`, also out of Task 3's scope.
  No dangling call site inside Task 3's actual scope.
- Build/test: `go build`/`go test` for the six touched packages plus `respawn` pass under the
  module's `go.work` (`cd services/atlas-channel/atlas.com/channel && go test ./channel/...
  ./note/... ./world/... ./compartment/... ./inventory/... ./transport/route/... ./respawn/...`
  → all `ok`). Note: `GOWORK=off go build ./...` from that module fails with a pre-existing,
  unrelated `missing go.sum entry ... atlas-env` error that exists identically on the pre-Task-3
  commit `1b1631c8a` too — not introduced by this range, and orthogonal to the six builders.

No findings for Task 3.

## Task 2 — `classify` subcommand

Files: `docs/tasks/task-263-backend-guideline-conformance/codemod/classify.go`,
`classify_test.go`, `main.go` (wiring only), and the six generated TSV/ledger outputs.

**Wiring** — `main.go:18` registers `"classify": runClassify` in the `subcommands` map exactly
as the plan's Task 2 required (Tasks 6/10/19 add the other three later). PASS.

**DOM-04 tier rules** (`ClassifyDOM04`, `classify.go:283-313`) — checked clause-by-clause
against the plan's table:
- `NO-RESTMODEL` when no type named exactly `RestModel` exists, evidence lists all
  `*RestModel`-suffixed type names — matches `findRestModelTypes` (`classify.go:318-340`).
- `C` when zero `Extract*` funcs — matches.
- `B1` when 2+ `Extract*` funcs, evidence `"<n> Extract funcs: <names>"` — matches format
  exactly.
- `B2` when the sole `Extract` is outside `rest.go`, or fails `isFlatExtract` — matches, checked
  in the plan's stated order (file check first, then flatness).
- `A` when the sole `Extract` is in `rest.go` and flat — matches.
- `isFlatExtract` (`classify.go:378-410`) implements the plan's literal "flatness" definition:
  single return statement, two results, second is bare `nil`, first is a `*ast.CompositeLit`
  (optionally behind `&`), every element a `KeyValueExpr` whose value is a selector or a
  single-arg call wrapping a selector. All nine `TestClassifyDOM04` table cases from the plan's
  Step 1 are present verbatim (`classify_test.go:64-248`) and pass (`go test -v` run below).

**Finding (non-blocking) — `isFlatFieldValue` does not check the selector is *on the
parameter*.** The plan's flatness definition says a field value must be "an `*ast.SelectorExpr`
**on the parameter**" (plan.md line ~139). `isFlatFieldValue` (`classify.go:412-429`) only
checks that `sel.X` is *some* `*ast.Ident`, without comparing it to the `Extract` function's
actual parameter name. A hypothetical `Extract` such as
`func Extract(rm RestModel) (Model, error) { return Model{a: somePackageVar.A}, nil }` would be
misclassified `A` (flat) even though the field is not derived from the request parameter.
I mechanically verified this is not currently exploited: a standalone AST walk over all 94
real `A`-tier `rest.go` `Extract` functions found zero selectors whose base identifier is not
the function's own parameter name (script run and discarded; no output = no violations). Given
this is a rule about a mechanical AST shape rather than a currently-triggered bug, and Task 2's
brief is explicitly to re-derive the *current* tiers (not to gate future code), this is
non-blocking — but worth tightening before Task 10's `transform` subcommand starts relying on
`A` meaning "safe to auto-collapse," since a future package added to the tree could exploit the
gap silently.

**DOM-01 FR-15 rules** (`classifyFR15`, `classify.go:481-500`) — matches the plan's three-way
table: `NO-MODEL-GO` is short-circuited before loading a package
(`classifyByModuleRoot`, `classify.go:160-166`, `os.Stat` on `model.go`); `RENAME` requires
exactly one `New*Builder` **and** `buildsPackageModel` true; everything else is
`SIBLING-EXEMPT`. `buildsPackageModel` (`classify.go:518-539`) correctly resolves the
constructor's return type's `Build()` method via `types.LookupFieldOrMethod`, unwraps a pointer
result, and requires the named result type be literally `Model` **and** declared in the same
package (`named.Obj().Pkg() == pkg.Types`) — correctly distinguishes a package's own `Model`
from an unrelated imported type of the same name. `TestClassifyFR15` covers both the sole/Model
case and the sole/non-Model ("differently named type") case from the plan.

**FILE-05 rules** (`classifyFILE05Inventory`, `classify.go:250-282`) — evaluated in the plan's
stated priority: `EXCLUDED-TREE` (`libs/atlas-packet/` prefix) → `ENTITY-BUILDER`
(`entity_builder.go` basename) → `HAS-BUILDER-GO` (sibling `builder.go` exists) → `RELOCATE`.
Matches the table exactly. `TestClassifyFILE05Inventory` covers `HAS-BUILDER-GO`, `RELOCATE`,
and `ENTITY-BUILDER`; it does not add a case for `EXCLUDED-TREE`, but that branch is a one-line
string-prefix check with negligible risk — non-blocking.

**Row/ledger completeness** — verified directly against the repo:
- `classify-dom04.tsv` / `classify-dom01-fr15.tsv` / `classify-file05.tsv` have exactly 185 / 69
  / 100 rows (`wc -l`), matching the plan's sum-check requirement exactly.
- Each of the three `*-ledger.tsv` files has the same row count as its source inventory file
  (185/69/100) with zero `SKIPPED` entries — every input line was classified, none vanished.

**Tests** — ran `GOWORK=off go test ./... -v` inside
`docs/tasks/task-263-backend-guideline-conformance/codemod`: all of `TestClassifyDOM04` (9
subtests), `TestClassifyFR15` (2 subtests), `TestClassifyFILE05Inventory`, and the pre-existing
`TestLedger` pass. These are genuinely new tests exercising newly-added functions
(`ClassifyDOM04`, `classifyFR15`, `classifyFILE05Inventory` did not exist before this range), so
they are not vacuously-passing coverage.

**Build artifact hygiene** — `docs/tasks/task-263-backend-guideline-conformance/codemod/atlas-task-263-codemod`
(the compiled binary) is `.gitignore`d and not tracked (`git ls-files` confirms only `.go`/
`.tsv`/`.mod`/`.sum` files are committed).

No blocking findings for Task 2.

## Not evaluable

- `report.go`/`load.go` (`Ledger`, `LoadModules`) are unchanged in this range (verified via
  `git diff --stat` returning no output for those two files) and were presumably reviewed under
  Task 1; this review used them as-is without re-auditing their internals.
- Could not run the full-repo `GOWORK=off go build ./...`/`go test ./...` for
  `services/atlas-channel` end-to-end because of a pre-existing, unrelated go.sum gap outside
  this range's diff; substituted targeted package builds/tests under the workspace `go.work`
  instead (see Task 3 section).

## Verdict

APPROVED_WITH_FINDINGS — one non-blocking finding (the flatness check's missing
parameter-identity comparison in `isFlatFieldValue`), currently harmless against the real
corpus but worth tightening before Task 10 (`transform`) leans on tier `A` for automated
rewrites.
