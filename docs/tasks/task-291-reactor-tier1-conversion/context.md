# task-291 — Implementation Context

Companion to [plan.md](plan.md). Everything here was read out of this worktree
at plan time; nothing is remembered.

## Key files and what they establish

| File | What it settles |
|---|---|
| `services/atlas-reactor-actions/atlas.com/reactor/script/executor.go:45-84` | The authoritative list of 11 dispatched operations |
| `.../executor.go:100-150` | `drop_items` reads `dropType`, `meso`, `mesoChance`, `mesoMin`, `mesoMax`, `minItems`; the `minMeso`/`maxMeso` fallback lives in two `else if` branches; defaults are `mesoChance=1, mesoMin=1, mesoMax=1, minItems=0`, `dropType="drop"` |
| `.../executor.go:181-215` | `spawn_monster` reads `monsterId` (required), `count` (default 1), `x`/`y` (default reactor position) |
| `.../executor.go:247-256` | `executeSprayItems` mutates `op.Params()["dropType"]="spray"` and delegates — identical parameter set |
| `.../executor.go:259-305` | `weaken_area_boss`, `move_environment`, `kill_all_monsters` are logging stubs |
| `.../evaluator.go` | Two condition types (`reactor_state`, `pq_custom_data`), six operators |
| `.../entity.go:31-59` | `jsonReactorScript` / `jsonRule` / `jsonCondition` / `jsonOperation` — the seed file's decode target. `TouchRules` and `Description` are `omitempty`; `Params` is `omitempty` |
| `.../processor.go:185-187` | `TouchRules()` falls back to `hitRules` when a script declares none |
| `.../subdomain.go:43-73` | `ReactorSubdomain.Build` iterates `HitRules` and `ActRules` only — see "Design corrections" below |
| `deploy/seed/gms/83_1/.../reactor-2001.json` | The byte conventions: 2-space indent, alphabetical keys, LF, one trailing newline; and the meso-shift regression |
| `deploy/seed/gms/83_1/.../reactor-9102002.json` | A bare `rm.dropItems()` emits an operation with only `"type"` — no `params` key |
| `deploy/seed/gms/83_1/.../reactor-9108000.json` | Condition key order (`operator`, `step`, `type`, `value`) and the guarded-rule-first ordering that `processor.go` depends on |
| `tools/catalog-lint/{go.mod,main.go,main_test.go,subdomains.go}` | The whole shape of a `tools/` Go module: own `go.mod` with `replace` back to `libs/`, a `lint(root) error` that accumulates `errs []string`, a `buildLint(t)` test helper that `go build`s the binary and execs it |
| `libs/atlas-seeder/jsonapi.go:9-80` | `Envelope`, `ParseEnvelope`, `ExtractEntityID` — the lint reuses these rather than re-parsing the envelope |
| `tools/verify.sh:213-243` | `touched()` and `changed_tool_suites()` — how a new gate step and its shell suite get selected |
| `tools/verify.sh:730-734` | The two-line `if touched … step … else skip … fi` idiom the new step copies |
| `tools/verify.sh:392-398` | `all_modules()` walks `services/` and `libs/` **only**; `tools/` modules are explicitly excluded from the Go sweep |
| `tools/verify.sh:883` | The precedent for running a non-swept module's Go tests as a named `step` |
| `tools/shell-guard.sh` | New `tools/*.sh` must pass `bash -n` and `shellcheck` at severity `error` |

## Decisions carried from the design, and why

- **A committed fail-closed generator, not 159 agent conversions and not a JS AST walker.** The census in design §3.1 shows 23 statement forms across all 159 bodies, ~8 of them structural. A hand-written matcher over that is smaller and more auditable than an AST walk, and it can abort on the first unrecognized line — which is the property that makes the output trustworthy. The plan makes that abort testable (Task 9's negative cases).
- **The lint, not the generator, is the durable gate.** `tier1-inventory.md` is a task-folder artifact and will not survive as a permanent build input. `reactor-seed-lint` depends only on the seed corpus and the schema, so it is what goes into `verify.sh`. `reactor-seed-gen -check` is a task-lifetime tool.
- **`additionalProperties: false` on the `drop_items`/`spray_items` param branches** turns FR-9 from a documentation change into an enforced one. Without it `"mesoRange": "15"` validates, because the top-level `params` accepts any string value. This is what makes the lint able to see the regression the corpus already had.
- **Assertions 3 and 4 live in the lint, not the schema.** `description` is legitimately optional on the REST resource, so requiring it in the schema would be wrong; it is a corpus rule. Cross-tenant byte identity is not expressible in JSON Schema at all.
- **`seedDirs` is a literal list in `emit.go`.** Adopting `NewFilesystemCatalogSourceWithShared` later (PRD §9.1) is then a change to one variable plus a `git rm`, not a rewrite.

## Design corrections made at plan time

Two claims in `design.md` did not survive contact with the code:

1. **"The generator ships with unit tests that run under the existing Go test sweep" (design §5.3) is false.** `tools/verify.sh:392-398` documents in its own comment that `all_modules()` deliberately excludes `tools/` modules — `tools/catalog-lint`'s Go tests have never run under `verify.sh`. Resolution: `tools/reactor-seed-lint_test.sh` (which `changed_tool_suites()` does run) invokes `go test ./...` in both new tool modules. Plan Task 6, Step 2.

2. **"existing `executor` tests updated for the removed fallback" (design §7) assumes a file that does not exist.** `services/atlas-reactor-actions/atlas.com/reactor/script/` has `evaluator_test.go`, `entity_test.go`, `builder_test.go`, `groups_test.go`, `processor_test.go`, `rest_test.go`, `provider_tenant_test.go`, `resource_pagination_test.go` — but no `executor_test.go`. The plan creates it (Task 13). The seam is `OperationExecutor.sagaP`, an unexported field of the one-method interface `atlas-reactor-actions/saga.Processor`; a same-package test substitutes a fake directly, with no new constructor and no `*_testhelpers.go`.

## Scope addition: Task 3

`ReactorSubdomain.Build` (`subdomain.go:43-73`) never iterates `attrs.TouchRules`, so a seed file's `touchRules` is silently dropped — while `entity.go:88-93` (the DB round-trip) and `rest.go:182-187` (the REST path) both handle it, and `entity_test.go:282` covers the entity path. FR-8 makes the schema advertise `touchRules` as a seedable key. Documenting a key the seeder ignores is worse than not documenting it, and the fix is the same six-line loop `entity.go` already runs. Task 3 adds it with a three-case table test. This is one task beyond the PRD's stated scope, deliberately: it is a prerequisite the branch can produce itself, not a follow-up.

## Sizing notes

Fourteen tasks. Three are deliberately outside the "≤ 6 files" heuristic and are noted here as the plan's format requires:

- **Task 4** lists eight files, but six of them are the new `tools/reactor-seed-lint` module created in one go (`go.mod`, `go.sum`, `main.go`, `schema.go`, `main_test.go`, `testdata/`) plus a one-line `go.work` edit and one read-only reference. Splitting a module's scaffold from its first assertion would produce a task with nothing testable in it.

- **Task 7** touches twelve files: eleven copies of the *same* mechanical edit to `reactor-2001.json` plus one new audit document. This is the repeated-mechanical-change exception — splitting it by tenant directory would make the byte-identity assertion impossible to check within a task.
- **Task 12** touches eleven directories and 1,749 files, but writes no source: its entire content is `go run ./tools/reactor-seed-gen` plus six assertions on the result. There is nothing to split.

Task 3 crosses into `services/atlas-reactor-actions` while Tasks 4-6 and 8-12 are confined to `tools/` and `deploy/seed`; Task 13 returns to the service. No task spans two services.

## Ordering constraints

FR-21 is the hard one: the legacy fallback (Task 13) may only be removed after `grep -rn 'minMeso\|maxMeso\|mesoRange\|"item"' deploy/seed/*/*/reactor-actions/` is empty, which Task 12 Step 6 establishes.

```
1 (skill)          ─┐
2 (schema) ─── 3 (touchRules seeding)
                    ├─ 4 ─ 5 ─ 6 (lint + wiring) ─ 7 (existing seeds) ─┐
8 ─ 9 ─ 10 ─ 11 (generator) ───────────────────────────────────────────┴─ 12 (generate) ─ 13 (fallback removal) ─ 14 (review + gate)
```

Tasks 1, 2+3, and 8-11 are mutually independent and can run in parallel. Task 6 Step 5 is expected to FAIL against the real corpus (reactor-2001 still carries legacy keys); Task 7 Step 6 is the first point the lint goes green.

## External dependency

`github.com/santhosh-tekuri/jsonschema/v6` — a new dependency, confined to `tools/reactor-seed-lint`'s own `go.mod` exactly as `tools/catalog-lint` confines its `atlas-seeder` dependency. **v6.0.3 is already in the local module cache** (`$GOPATH/pkg/mod/cache/download/github.com/santhosh-tekuri/jsonschema/v6/@v/v6.0.3.zip`), so `go mod tidy` resolves offline. Its API (`UnmarshalJSON`, `NewCompiler`, `AddResource`, `Compile`, `Validate`) is this repo's first use — Task 4 Step 5 tells the implementer to confirm the signatures with `go doc` rather than trust the plan's code block.

## Reference checkout

Task 7's audit of the twelve PQ seeds reads `<Cosmic>/scripts/reactor/<id>.js` from a sibling reference-server clone outside this repository. It is present on the plan author's machine and holds all 292 scripts including `9102002`–`9102007` and `9108000`–`9108005`. Per CLAUDE.md, its absolute path must never be written into a committed file — the audit document refers to sources as `<Cosmic>/scripts/reactor/<id>.js`.

## Corpus facts measured at plan time

- `tier1-inventory.md` has exactly **159** `### \`<id>\`` headings.
- **6** sections contain a `function hit()`: `2119000`–`2119003` (all four guarded), `2612004`, `2619000`.
- **2** sections contain a `for` loop: `2201001` (×3) and `2511001` (×6).
- **5** sections touch `eim`/`getEventInstance`: `2002003`, `2008006`, `2511000`, `2512001`, `9208009`.
- **15** sections say `*(no comment in source)*`; roughly **25 more** clean to nothing or to degenerate text, so the override table is ~40 entries. Ids observed to need one include `1022001`, `1032000`, `1202000`, `1402000`, `200000`–`200009`, `2052001`, `2112015`, `2119004`–`2119006`, `2229009`, `6741001`, `6741015`, `6742014`, `6802000`, `6802001`, `8001000`.
- `reactor-2001.json` is the **only** file in the whole seed tree matching `minMeso|maxMeso|mesoRange|"item"`.
- Each of the eleven tenant directories currently holds **13** reactor files; each must hold **172** after Task 12.

## plan-lint outcome

`tools/plan-lint.sh docs/tasks/task-291-reactor-tier1-conversion/plan.md` exits **0**: no F1, F2 or F3 errors.

- **F4 (advisory, 3 hits):** Tasks 4, 7 and 12 — justified under "Sizing notes" above.
- **F5 (advisory, 4 hits):** `NewCompiler`, `AddResource`, `Compile`, and `dropItems`. The first three are `github.com/santhosh-tekuri/jsonschema/v6`, this repo's first use of that library; all five signatures the plan calls were read from the cached module source and are exact for v6.0.3 — `UnmarshalJSON(io.Reader) (any, error)` (`loader.go:255`), `NewCompiler() *Compiler` (`compiler.go:21`), `(*Compiler).AddResource(string, any) error` (`compiler.go:121`), `(*Compiler).Compile(string) (*Schema, error)` (`compiler.go:178`), `(*Schema).Validate(any) error` (`validator.go:15`). `dropItems` is a JavaScript method name quoted in a Go doc comment, not a Go symbol.

Nothing invented; every advisory was checked rather than waived.
