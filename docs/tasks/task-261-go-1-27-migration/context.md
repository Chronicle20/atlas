# Go 1.27 Migration — Implementation Context

Companion to `plan.md`. Consumes `prd.md` (v1), `design.md` (v1), `pin-inventory.md`.
Branch `task-261-go-1-27-migration`, branch point `855fef4d1`, design commit `bc870d75c`.

---

## The three values everything else derives from

```
GO_VERSION=1.27.0
ALPINE_VERSION=3.24
GOLANGCI_LINT_VERSION=v2.13.1
```

Declared once in `tools/toolchain.versions` (Task 1). Every other occurrence in the
repo is a duplicate that no file format can cross-reference, machine-checked by
`tools/toolchain-pin-guard.sh` (Task 7).

## Key files

| File | Role |
|---|---|
| `tools/toolchain.versions` | **New** (git mv from `tools/lint.versions`). The source of truth. |
| `tools/toolchain-pin-guard.sh` | **New**. Pure bash. Enumerates `go.mod` via `git ls-files`; checks 7 site classes; `--selftest`. |
| `tools/lint.sh:19-20,38` | Sources the pin file; usage text names it. |
| `.claude/hooks/format-on-write.sh:28-29` | Sources the pin file with `2>/dev/null \|\| exit 0` — a broken path here is **silent**. |
| `.golangci.yml:12` | Comment reference only; no version literal. |
| `.github/workflows/pr-validation.yml:41,478` | `GO_VERSION` env pin + the lint-tools cache key `hashFiles()`. |
| `Dockerfile:17-18,20,94` | Global ARGs, the builder `FROM`, and the synthesized `go.work` directive. |
| `services/atlas-kafka-precreate/Dockerfile:9-10` | Live (its bake target's context is the service dir), flat module, synthesizes nothing. |
| `docker-bake.hcl:8-14` | `GO_VERSION` / `ALPINE_VERSION` variable defaults. |
| `services/atlas-renders/Dockerfile` | Dead; deleted in Task 4. |
| `go.work:1` | Workspace directive. 95 `use` entries, unchanged. |
| `renovate.json:18-99` | 11 `packageRules`; Task 9 appends two, making 13. |
| `README.md:79` | Prerequisites Go row — a checked site after Task 6. |
| `tools/verify.sh:94,109,147,431-437` | `step()`, `skip()`, `touched()` helpers; the insertion point. |

## Decisions carried from design.md

1. **D1 — rename, don't extend.** `tools/lint.versions` → `tools/toolchain.versions`.
   Extending in place would have cost zero consumer edits but left a file named
   `lint.versions` declaring the Go and Alpine version of every container image.
   Cost of the rename: five mechanical, greppable edits, paid once.
2. **D2 — bash guard, not a Go analyzer.** The four `tools/*guard` analyzers exist
   because their invariants need type information. This one compares string literals
   in `go.mod`, YAML, HCL, and Dockerfiles. Bash also means the CI job needs no
   `setup-go` — which matters because this guard's purpose is to be the thing that
   still works when the Go pin is wrong.
3. **D3 — CI needs its own job.** CI does not run `tools/verify.sh`; it enumerates
   each shell guard as a job. FR-6.5's conditional resolves to its second branch.
4. **D4 — `go mod edit`, no `go mod tidy`.** See "Landmines" below.
5. **D5 — `Dockerfile:94` derived, not pinned.** `printf 'go %s\n\nuse (\n' "${GO_VERSION}"`
   with a stage-local `ARG GO_VERSION` re-declaration. A pin site that cannot drift
   beats one the guard must police — especially one that would require parsing a
   `printf` format string inside a line-continued `RUN`.
6. **D6 — Renovate fixed by ordering.** Later rules win; the overrides go last.

## Dependencies between tasks

```
Task 1 (pin file) ──┬─> Task 2 (103 go.mod + go.work) ──> Task 3 (lint/vet blast radius)
                    ├─> Task 4 (containers)
                    ├─> Task 5 (CI pins)
                    └─> Task 6 (README)
                                   ↓  (all pin sites must be at 1.27.0 first)
                            Task 7 (write the guard) ──> Task 8 (wire it)
Task 9 (renovate) — independent of everything
                            ↓
                    Task 10 (acceptance sweep) — depends on all
```

**Task 3 runs third, not last.** PRD open question 3 (the volume of new Go 1.27
`go vet` and golangci-lint v2.13.1 diagnostics) is the one item that can materially
change the size of this task, and it is only answerable by running the sweep.
Measuring it while the branch is still small keeps the escalation cheap.

**Task 7 must come after 2/4/5/6.** The guard is written against a tree where every
pin site already agrees; writing it earlier means debugging real violations and guard
bugs simultaneously.

## Landmines

- **`go mod tidy` cannot run standalone on a service module in this repo.** Proven on
  the *unmodified* tree (design §1.3): `atlas-account` reaches `atlas-env`
  transitively through `atlas-kafka` and carries no `replace` for it, so only
  workspace mode resolves it — and `go mod tidy` deliberately ignores `go.work`.
  There is no flag that fixes this. FR-1.4 as written is not executable; `plan.md`
  Task 2 uses `go mod edit` + `go work sync` instead.
- **Expected `go.sum` churn is zero.** The `go` directive selects the language
  version and the pruning mode; pruning has been constant since 1.17. A `go.sum`
  delta means something *other than* the directive moved — stop, do not commit.
- **`go.work.sum` may legitimately shrink.** It currently carries stale
  `github.com/golangci/golangci-lint/v2 v2.12.2` entries at `:273-274` that no
  tracked `go.mod` requires (residue from a past `go install`). `go work sync` may
  drop them. Benign; mention in the PR description.
- **Order matters: modules before workspace.** Go rejects a workspace whose `go`
  directive is below a member's, so editing `go.work` first leaves the tree
  transiently unbuildable and makes any mid-sweep `go build` misleading.
- **A global `ARG` is invisible inside a build stage.** If Task 4 Step 2's
  re-declaration is omitted, `${GO_VERSION}` expands to the empty string and the
  synthesized workspace gets `go \n` — failing every go-service image build. Only an
  actual `docker buildx bake` catches this, which is why Task 4 Step 8 builds one.
- **The format-on-write hook fails silently.** `.claude/hooks/format-on-write.sh:29`
  sources the pin file with `2>/dev/null || exit 0`. A rename typo there disables Go
  formatting for every future edit and a green `verify.sh` never notices. Task 1
  Step 6 exists solely for this.
- **`.github/actions/go-test/action.yml` has more than one `default:` key.**
  `race-detection` has one too. The guard must scope its match to the `go-version:`
  input's block, not the first `default:` in the file.
- **Eight in-scope modules are outside `go.work`.** All eight `tools/*guard*`
  analyzer modules. 103 in-scope − 95 workspace `use` entries = 8. A workspace-only
  sweep misses them; `git ls-files '*go.mod'` does not.
- **111 tracked `go.mod` files, 103 in scope.** The 8-file difference is exactly
  `tools/cideps/testdata/`. If `git ls-files '*go.mod' 'go.mod' | wc -l` is not 111,
  the module set moved since the inventory and the plan's counts need re-deriving.

## Deliberately oversized task (F4)

**Task 2 touches 104 files** (103 `go.mod` + `go.work`). It is not split, because it
is a single mechanical edit — `go mod edit -go=1.27.0` — applied uniformly with zero
per-file judgement, and because the intermediate state of a partial split is exactly
the drift this task exists to eliminate: a tree where some modules declare 1.27.0 and
others do not fails workspace load, so every sub-task's `go build` verification would
be meaningless until the last one landed. The step's verification is a whole-tree
tally, not a per-file inspection, so the implementer's tool-call count stays flat in
the file count.

Task 4 touches 4 files and one deletion; Task 5 touches 6; every other task touches
5 or fewer. Task 5's six files are one-line quoted-scalar replacements from a table.

## Verification commands

```bash
tools/toolchain-pin-guard.sh              # exit 0; the durable deliverable
tools/toolchain-pin-guard.sh --selftest   # proves the guard can fail
tools/lint.sh --check                     # golangci-lint v2.13.1, whole tree
go build ./...                            # workspace mode, repo root
cd tools/cideps && go test ./...          # AC-13: fixtures untouched and passing
tools/verify.sh                           # FLAGLESS. --quick/--no-docker do not count.
```

## What is explicitly NOT done

- No `toolchain` directive anywhere (AC-2).
- No Go 1.27 language feature, stdlib API, or `GOEXPERIMENT` adoption.
- No dependency upgrade beyond `go work sync` churn.
- No `tools/cideps` fixture bump, and no edit to `tools/cideps/graph_test.go:14` or
  `tools/plan-context_test.sh:50`.
- No Renovate `customManagers` for the pin file — it would open a PR editing the pin
  file alone, which the guard would immediately fail because the 103 `go.mod` files
  had not moved. A recurring red PR whose only resolution is the manual sweep the
  guard already forces.
- No change to `.github/workflows/reconcile-bump-prs.yml` — it reconciles image-tag
  bump PRs and encodes no toolchain assumption (design §1.8).
- No teaching of `docker-bake.hcl` or the Dockerfiles to read the pin file — neither
  `go.mod`/`go.work` directives nor Dockerfile `ARG` defaults can read an external
  file at all, so the guard would still be required for the majority of sites.
- No `services/atlas-ui` Node work.
- No rewriting of the historical `docs/tasks/task-171-*` artifacts that reference
  `tools/lint.versions`.
