# Task 20 — Full verification sweep

Run from the task worktree root (`<repo-root>/.worktrees/task-194-packet-definition-matrix`)
on branch `task-194-packet-definition-matrix`, merge-base
`31c7a664f975e8fadcd2e0e4e893427bddc340d9`.

Every command below was actually executed; output is pasted verbatim (trimmed
to the meaningful head/tail where noted). Nothing in this document is
paraphrased or inferred.

## 1. Go gates — atlas-configurations

```
$ cd services/atlas-configurations/atlas.com/configurations
$ go build ./... && go vet ./... && go test -race ./...
```

Result: **PASS**

```
=== go build ===
(no output — clean)
=== go vet ===
(no output — clean)
=== go test -race ===
Go test: 185 passed in 35 packages
```

(`go test` output above is via the project's `rtk` output filter; verbatim
pass/package counts as reported.)

Re-run **after** the lint fix (see §6), to prove goimports/gofumpt did not
change behaviour:

```
=== go build ===
(no output — clean)
=== go vet ===
(no output — clean)
=== go test -race ===
Go test: 185 passed in 35 packages
```

Identical: 185 passed / 35 packages, before and after.

## 2. Go gates — packet-audit

```
$ cd tools/packet-audit
$ go build ./... && go vet ./... && go test -race ./...
```

Result: **PASS**

```
=== go build ===
(no output — clean)
=== go vet ===
(no output — clean)
=== go test -race ===
Go test: 440 passed in 14 packages
```

Re-run after the lint fix: identical — `Go test: 440 passed in 14 packages`.
(No file under `tools/packet-audit` was touched by the lint fix, so this
suite was never at risk, but it was re-run anyway per the sweep.)

## 3. Container build — atlas-configurations

Mandatory because Task 3 added a `require` to `go.mod`; `go build` against
the workspace `go.work` does not catch a missing `COPY libs/...` line in the
shared Dockerfile.

```
$ docker buildx bake atlas-configurations
```

Result: **PASS** (exit 0)

Verbatim tail:

```
#57 [build-env 48/51] RUN MOD_DIR=$(ls -d services/atlas-configurations/atlas.com/*/ | head -1) ... go.work
#57 DONE 0.4s

#58 [build-env 49/51] RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build MOD_DIR=... && go build -C "$MOD_DIR" -o /server
#58 DONE 3.1s

#59 [build-env 50/51] RUN ... cp "${MOD_DIR}config.yaml" /app/config.yaml ...
#59 DONE 0.7s

#60 [build-env 51/51] RUN set -e; for src in seed-data drops data scripts conversations shops party-quests configurations; do ... cp -r ... /app/${src}; done
#60 DONE 0.7s

#61 [stage-1  2/13] RUN apk add --no-cache libc6-compat
#61 CACHED

#62-#71 COPY --from=build-env /server /, config.yaml, seed-data, drops, data, scripts, conversations, shops, party-quests, configurations
(all DONE)

#72 exporting to image
#72 exporting layers 1.2s done
#72 exporting manifest ... done
#72 naming to docker.io/library/atlas-configurations:local done
#72 unpacking to docker.io/library/atlas-configurations:local 0.3s done
```

`echo $?` immediately after a repeat run captured `EXIT=0`. `libs/atlas-opcodes`
was already present in the `COPY` list (no Dockerfile edit needed — confirmed
by the build succeeding without modification).

## 4. Template guards

```
$ tools/template-opcode-order-guard.sh
OK: 22 template arrays are in ascending opcode order.
(EXIT=0)

$ tools/template-duplicate-binding-guard.sh
OK: 22 template arrays carry no duplicate (name, opCode) binding.
(EXIT=0)
```

Result: **PASS** — matches the expected messages exactly.

## 5. Repo-wide guards

```
$ tools/redis-key-guard.sh
```
Enumerated every service module (`rediskeyguard: <path>` per module, ~65
modules), no violations reported. `(EXIT=0)`

```
$ tools/goroutine-guard.sh
```
Self-test passed (`ok  	github.com/Chronicle20/atlas/tools/goroutineguard	(cached)`),
then enumerated every service + lib module (`goroutineguard: <path>`), no
violations reported. `(EXIT=0)`

Result: **PASS** for both.

`tools/service-registration-guard.sh` — **not run**, per plan. Verified the
skip condition directly:

```
$ git diff --name-only 31c7a664f975e8fadcd2e0e4e893427bddc340d9..HEAD | grep -E 'services\.json|deploy/k8s|docker-bake|go\.work|db-bootstrap'
(no output)
```

None of `services.json`, `deploy/k8s`, `docker-bake.hcl`, `go.work`, or
`tools/db-bootstrap.sh` changed on this branch. Skip confirmed, not assumed.

## 6. Lint and format — `tools/lint.sh --check`

**First run (before any fix) — FAILED.**

```
$ source ~/.nvm/nvm.sh && nvm use 22
Now using node v22.22.2 (npm v10.9.7)
$ tools/lint.sh --check
...
services/atlas-configurations/atlas.com/configurations/templates/socket_validation_test.go:148:8: QF1011: could omit type configsocket.Input from declaration; it will be inferred from the right-hand side (staticcheck)
	var _ configsocket.Input = in
	      ^
services/atlas-configurations/atlas.com/configurations/tenants/socket_validation_test.go:137:8: QF1011: could omit type configsocket.Input from declaration; it will be inferred from the right-hand side (staticcheck)
	var _ configsocket.Input = in
	      ^
2 issues:
* staticcheck: 2
lint.sh: LINT FAIL — services/atlas-configurations/atlas.com/configurations
...
services/atlas-ui/src/lib/socket/matrix.ts
  213:9  error  The value assigned to 'cmp' is not used in subsequent statements  no-useless-assignment
...
✖ 7 problems (1 error, 6 warnings)

lint.sh: UI LINT FAIL — services/atlas-ui

lint.sh: FAIL — 3 failing target(s):
lint.sh:   lint:services/atlas-configurations/atlas.com/configurations
lint.sh:   ui:prettier
lint.sh:   ui:eslint
lint.sh: run 'tools/lint.sh' (fix mode) locally, then commit the result.
EXIT=1
```

**What was fixed:**

1. Ran `tools/lint.sh` (fix mode, no flags). It auto-fixed:
   - The 2 `staticcheck` QF1011 hits (`socket_validation_test.go` ×2) by
     removing the now-redundant explicit interface type from
     `var _ configsocket.Input = in` → `_ = in`, and dropping the
     now-unused `configsocket` import in both files.
   - Prettier reformatting drift across **29 atlas-ui TS/TSX files** under
     `src/components/features/socket/` and `src/lib/socket/` (introduced by
     Tasks 8–18; flagged but explicitly left unfixed by Task 19 to keep its
     diff scoped).
2. One ESLint finding was **not** auto-fixable: `no-useless-assignment` on
   `src/lib/socket/matrix.ts` (`let cmp = 0;` — the initializer was never
   read, since every branch of the following `if/else if/else` assigns it
   before use). Fixed by hand: `let cmp = 0;` → `let cmp: number;`
   (`services/atlas-ui/src/lib/socket/matrix.ts:209`).

**Correction (post-review):** golangci-lint's staticcheck `--fix` for QF1011
did more damage than intended. `var _ configsocket.Input = in` was a genuine
compile-time assertion that `ToValidationInput` returns the SHARED
`configsocket.Input` type, not a tree-local look-alike — exactly the drift
Task 5's shared-adapter design exists to prevent. The auto-fix reduced it to
`_ = in`, which type-checks against any type whatsoever and asserts nothing,
while leaving the comment above it (`// Compile-time proof the adapter
returns the shared package's type.`) claiming a guarantee that no longer
existed.

Fixed in both `templates/socket_validation_test.go` and
`tenants/socket_validation_test.go`: restored the `configsocket` import, and
replaced `_ = in` with a call to a new unexported helper,
`func assertSharedInput(configsocket.Input) {}`, invoked as
`assertSharedInput(in)`. A function-parameter check carries the same
compile-time guarantee as the original `var` form, but QF1011 only fires on
variable declarations, not call expressions, so it will not be re-flagged.

**Proof the restored assertion is live** (performed once for each tree,
then fully reverted — confirmed via `git diff` showing empty on
`adapter.go`/`processor.go` afterward):

1. In `templates/socket/adapter.go`, temporarily added
   `type driftProofLocalInput configsocket.Input` and changed
   `ToValidationInput`'s return type to it (converting the return value).
   `go build ./...` failed — but at the *production* callsite
   (`templates/processor.go:168`, `configsocket.Validate(...)`), not yet at
   the test:
   ```
   templates/processor.go:168:31: cannot use socket.ToValidationInput(rm) (value of struct type "atlas-configurations/templates/socket".driftProofLocalInput) as "atlas-configurations/socket".Input value in argument to configsocket.Validate
   ```
2. To isolate the test assertion specifically, temporarily wrapped the
   production callsite in an explicit conversion
   (`configsocket.Validate(configsocket.Input(socket.ToValidationInput(rm)))`)
   so production tolerated the drift. With that in place, `go build ./...`
   passed (exit 0), but `go vet ./templates/...` (which type-checks test
   files) failed exactly at the restored assertion:
   ```
   templates/socket_validation_test.go:157:20: cannot use in (variable of struct type "atlas-configurations/templates/socket".driftProofLocalInput) as "atlas-configurations/socket".Input value in argument to assertSharedInput
   ```
3. Repeated both steps for the `tenants` tree with identical results
   (`tenants/processor.go:218`, then
   `tenants/socket_validation_test.go:146:20: cannot use in ... in argument to assertSharedInput`).
4. Reverted all four temporary edits (`templates/socket/adapter.go`,
   `templates/processor.go`, `tenants/socket/adapter.go`,
   `tenants/processor.go`) — `git diff` on all four is empty against the
   committed lint-fix state.

This confirms `assertSharedInput` independently fails the build the moment
either adapter's return type stops being `configsocket.Input`, regardless
of whether the production callsite happens to still compile.

**Second `--check` run — GREEN:**

```
$ source ~/.nvm/nvm.sh && nvm use 22
$ tools/lint.sh --check
...
✖ 6 problems (0 errors, 6 warnings)

lint.sh: OK
EXIT=0
```

The 6 remaining warnings are pre-existing `react-hooks/incompatible-library`
/ `react-hooks/exhaustive-deps` warnings on files this branch never touches
(`CreateBanDialog.tsx`, `ApplyPresetDialog.tsx`, `CreateTenantDialog.tsx`,
`AccountsPage.tsx`, `QuestsPage.tsx`) — warnings, not errors, and `lint.sh`
reports `OK` with them present. They predate this branch.

**Third `--check` run (after the `assertSharedInput` correction) — GREEN,**
after one transient false failure caused by unrelated concurrent activity:

A first re-run reported `lint:libs/atlas-routine` and
`lint:services/atlas-character/atlas.com/character` as failing — both
paths this branch never touches. Inspecting the raw log showed the actual
cause was `Error: parallel golangci-lint is running` /
`The command is terminated due to an error: parallel golangci-lint is
running` for both, i.e. lock contention against golangci-lint invocations
running concurrently in *other* worktrees on the same machine (confirmed via
`ps aux` + `/proc/<pid>/cwd`: the concurrent `golangci-lint fmt --diff`
processes were rooted in `.worktrees/task-195-foreign-disease-mobskill` and
`.worktrees/task-196-npc-info-default-icon`, not this worktree). Re-running
`tools/lint.sh --check` alone, with no other lint.sh process active per
`ps aux`, produced:

```
$ source ~/.nvm/nvm.sh && nvm use 22
$ tools/lint.sh --check
...
✖ 6 problems (0 errors, 6 warnings)

lint.sh: OK
EXIT=0
```

Same 6 pre-existing warnings as before, 0 errors, `lint.sh: OK`.

**Files changed by the lint fix + the one hand-fix + the correction:**
still 31 unique file paths total (the correction is additional edits to
the same 2 Go test files already listed, not new paths) —
2 Go test files (`templates/socket_validation_test.go`,
`tenants/socket_validation_test.go`) and 29 atlas-ui files (all under
`src/components/features/socket/`, `src/lib/hooks/api/`, `src/lib/socket/`,
`src/services/api/__tests__/`).

Result: **PASS** (green; the correction is documented above and does not
change the file-count or the atlas-ui portion of the diff).

## 7. Post-lint-fix re-verification

Per the task's honesty requirement — prove the lint rewrite (goimports /
gofumpt / Prettier reorder, plus the one hand-fix) did not change behaviour.

**Go — atlas-configurations:**
```
go build ./... && go vet ./... && go test -race ./...
Go test: 185 passed in 35 packages
```
Same as §1's pre-fix run: **185 passed / 35 packages, unchanged.**

Re-confirmed once more after the `assertSharedInput` correction (§6):
same command, same result — `Go test: 185 passed in 35 packages`, `go
build`/`go vet` clean. The restored assertion adds one unexported no-op
helper function and one call per tree; it does not change any test's
behavior or count.

**Go — packet-audit:**
```
go build ./... && go vet ./... && go test -race ./...
Go test: 440 passed in 14 packages
```
Same as §2's pre-fix run: **440 passed / 14 packages, unchanged.**

**atlas-ui — `npm test`:**
```
$ source ~/.nvm/nvm.sh && nvm use 22
$ cd services/atlas-ui && npm test

 RUN  v4.1.10 <repo-root>/.worktrees/task-194-packet-definition-matrix/services/atlas-ui

 Test Files  208 passed (208)
      Tests  1659 passed (1659)
   Start at  13:06:28
   Duration  26.21s
EXIT=0
```

Baseline for comparison: Task 19's report (`.superpowers/sdd/plan/task-19-report.md`)
recorded the branch's test suite at **208 test files / 1659 tests passed**
as its own final state (after Task 19's fix round), and explicitly flagged
both issues this task fixed (the ~26-file Prettier drift and the
`matrix.ts:213` `no-useless-assignment` error) as pre-existing and
out-of-scope for Task 19. **208/1659 before, 208/1659 after — identical.**

**atlas-ui — `npm run build`:**
```
$ npm run build
...
✓ built in 1.42s
[plugin builtin:vite-reporter]
(!) Some chunks are larger than 500 kB after minification. [pre-existing warning, ConversationEditorPanel — unrelated to this task]
EXIT=0
```

`tsc -b` (part of `npm run build`) reported no type errors. Build clean.

Result: **PASS**, no regression from the lint fix.

## 8. Frontend gates (restated — already covered by §6/§7)

`npm test` and `npm run build` are documented above (§7): both clean,
208/1659 and exit 0 respectively.

## 9. Row-count evidence (PRD §10 acceptance: 141 handlers / 219 writers)

Computed directly from the eleven committed seed templates (real data, not
the running app — the row set per FR-2.5 is the union of Defined and
Unsupported names across selected Templates for the active mode):

```
$ ls services/atlas-configurations/seed-data/templates/*.json | wc -l
11

$ jq -s '[.[] | .socket.handlers[].handler, (.socket.unsupported.handlers // [])[]] | unique | length' \
    services/atlas-configurations/seed-data/templates/*.json
141

$ jq -s '[.[] | .socket.writers[].writer, (.socket.unsupported.writers // [])[]] | unique | length' \
    services/atlas-configurations/seed-data/templates/*.json
219
```

Matches the PRD's acceptance criteria (§10) exactly: **141 handler rows,
219 writer rows** across the eleven seed templates.

Per-template raw counts (before dedup / union), for reference:

| Template | handlers | writers |
|---|---|---|
| gms_12_1 | 24 | 42 |
| gms_48_1 | 81 | 62 |
| gms_61_1 | 115 | 153 |
| gms_72_1 | 124 | 160 |
| gms_79_1 | 125 | 196 |
| gms_83_1 | 133 | 208 |
| gms_84_1 | 134 | 209 |
| gms_87_1 | 122 | 206 |
| gms_92_1 | 47 | 65 |
| gms_95_1 | 129 | 214 |
| jms_185_1 | 112 | 198 |

The interactive, browser-driven acceptance criteria in PRD §10 (sidebar
rendering, baseline reorder, opcode search normalisation, dialog mutations
surviving reload, etc.) require a running app against a seeded environment
and are **not covered by this document** — this task's scope, as directed,
is the CLAUDE.md gate sweep plus the branch-level invariants below, not a
live-app UI walk.

## 10. Branch-level invariants

**No literal home/absolute paths in committed files:**

```
$ git diff 31c7a664f975e8fadcd2e0e4e893427bddc340d9..HEAD -- . ':!.superpowers' | grep -n '/home/[a-z]'
(no output — exit 1, i.e. no match)
```

**No new `// TODO` / `FIXME` / stub / 501 introduced in code:**

```
$ git diff 31c7a664f975e8fadcd2e0e4e893427bddc340d9..HEAD -- 'services/atlas-configurations/atlas.com/*' 'services/atlas-ui/src/*' 'tools/packet-audit/*' \
    | grep -nE '^\+.*(TODO|FIXME)'
(no output — exit 1, i.e. no match)
```

(A broader, unscoped grep also matched `docs/tasks/.../prd.md` prose that
*talks about* the "No TODO" rule, and IDA function names like `sub_50120A`
that happen to contain the substring `501` — both false positives, not
findings, confirmed by re-running scoped to actual code paths above.)

**No new frontend dependency:**

```
$ git diff 31c7a664f975e8fadcd2e0e4e893427bddc340d9..HEAD -- services/atlas-ui/package.json services/atlas-ui/package-lock.json
(no output — empty diff)
```

**Four deleted form files gone, no importers:**

```
$ git diff --name-status 31c7a664f975e8fadcd2e0e4e893427bddc340d9..HEAD -- \
    services/atlas-ui/src/pages/templates-handlers-form.tsx \
    services/atlas-ui/src/pages/templates-worlds-form.tsx \
    services/atlas-ui/src/pages/templates-writers-form.tsx \
    services/atlas-ui/src/pages/tenants-handlers-form.tsx \
    services/atlas-ui/src/pages/tenants-writers-form.tsx \
    services/atlas-ui/src/pages/templates-properties-form.tsx
D	services/atlas-ui/src/pages/templates-handlers-form.tsx
M	services/atlas-ui/src/pages/templates-properties-form.tsx
M	services/atlas-ui/src/pages/templates-worlds-form.tsx
D	services/atlas-ui/src/pages/templates-writers-form.tsx
D	services/atlas-ui/src/pages/tenants-handlers-form.tsx
D	services/atlas-ui/src/pages/tenants-writers-form.tsx
```

Exactly the four FR-7.4 forms are deleted (`templates-handlers-form.tsx`,
`templates-writers-form.tsx`, `tenants-handlers-form.tsx`,
`tenants-writers-form.tsx`); `templates-worlds-form.tsx` and
`templates-properties-form.tsx` are modified, not removed (they back
unrelated pages — `TemplatesCharacterTemplatesPage`/`TemplatesCharacterPresetsPage`
— out of FR-7.4's scope).

```
$ grep -rn "templates-handlers-form\|templates-writers-form\|tenants-handlers-form\|tenants-writers-form" \
    services/atlas-ui/src --include="*.tsx" --include="*.ts"
(no output — exit 1, i.e. no match)
```

No remaining import of any of the four deleted files.

## Summary

| # | Gate | Result |
|---|---|---|
| 1 | Go build/vet/test -race — atlas-configurations | PASS (185/35, before and after) |
| 2 | Go build/vet/test -race — packet-audit | PASS (440/14, before and after) |
| 3 | `docker buildx bake atlas-configurations` | PASS (exit 0) |
| 4 | `template-opcode-order-guard.sh` | PASS |
| 5 | `template-duplicate-binding-guard.sh` | PASS |
| 6 | `redis-key-guard.sh` | PASS |
| 7 | `goroutine-guard.sh` | PASS |
| 8 | `service-registration-guard.sh` | SKIPPED (verified: no matching files changed) |
| 9 | `tools/lint.sh --check` | PASS (failed first, fixed, green; one auto-fix was later corrected — see §6 — and re-confirmed green) |
| 10 | `npm test` (atlas-ui) | PASS (208/1659, before and after) |
| 11 | `npm run build` (atlas-ui) | PASS (exit 0) |
| 12 | Home-path leak check | PASS (none found) |
| 13 | TODO/FIXME/stub/501 check | PASS (none found; false positives noted) |
| 14 | New frontend dependency check | PASS (empty diff) |
| 15 | Deleted form files + no importers | PASS |

All gates required by this task are green. The fixes applied were the lint
drift (31 files: 2 Go, 29 TS/TSX) plus the associated re-verification above,
and a post-review correction (§6) restoring a real compile-time assertion
that one of the lint auto-fixes had reduced to a no-op — both committed
alongside this record.
