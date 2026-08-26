# Task 3 evidence: Go 1.27 / golangci-lint v2.13.1 blast radius (AC-9)

This records the measured lint/vet blast radius after the toolchain and
module-directive pins (Tasks 1-2) landed, the fix applied, and the
zero-finding re-run that closes the gate.

## 1. golangci-lint binary version

Command:

```
.cache/tools/bin/golangci-lint-v2.13.1 version
```

Output:

```
golangci-lint has version 2.13.1 built with go1.27.0 from 6d2288e0 on 2026-08-20T14:28:34Z
```

Confirms the v2.13.1 binary — pinned in `tools/toolchain.versions` by Task 1 —
is the one that ran every check below.

## 2. Step 1 — first measurement: `tools/lint.sh --check`

Command:

```
tools/lint.sh --check 2>&1 | tee /tmp/t261-lint-check.log; echo "LINT_EXIT=${PIPESTATUS[0]}"
```

(Run as `tools/lint.sh --check --go`, redirecting instead of piping through
`tee` — the sandbox's worktree-isolation guard rejects the piped form; the
exit code was captured with a separate `echo $?` appended to the same
redirected log, which is exit-code-equivalent.)

Final exit code: **`LINT_EXIT=1`**

Full finding list (the only non-`0 issues.` module in the whole-tree walk):

```
services/atlas-channel/atlas.com/channel/character/processor_test.go:246:8: QF1011: could omit type func(character.Model) character.Model from declaration; it will be inferred from the right-hand side (staticcheck)
	var _ func(character.Model) character.Model = mock.NewMockProcessor().PartyDecorator
	      ^
1 issues:
* staticcheck: 1
lint.sh: LINT FAIL — services/atlas-channel/atlas.com/channel
```

Every other module (90 of 91 discovered `go.mod` roots) reported `0 issues.`
for the lint layer.

Separately, the formatter layer (measured first via `tools/lint.sh --fmt --go`,
the fast fix-mode path the controller directed) found the 13 modules the
Task 2 gate had already identified, all one gofumpt rule: a blank line is now
required between a single-line function declaration and a following
multi-line function declaration.

```
fmt:libs/atlas-packet
fmt:services/atlas-ban/atlas.com/ban
fmt:services/atlas-cashshop/atlas.com/cashshop
fmt:services/atlas-channel/atlas.com/channel
fmt:services/atlas-consumables/atlas.com/consumables
fmt:services/atlas-events/atlas.com/events
fmt:services/atlas-inventory/atlas.com/inventory
fmt:services/atlas-login/atlas.com/login
fmt:services/atlas-monster-book/atlas.com/monster-book
fmt:services/atlas-npc-conversations/atlas.com/npc
fmt:services/atlas-query-aggregator/atlas.com/query-aggregator
fmt:services/atlas-rankings/atlas.com/rankings
fmt:services/atlas-summons/atlas.com/summons
```

`tools/lint.sh --fmt --go` (fix mode) rewrote all 13 modules in place; the
`.go` diff in this commit is exactly that formatter output (blank-line
insertion / redundant-paren removal), nothing hand-edited.

## 3. Step 2 — `go vet` blast radius

`go vet ./...` does not run from the repo root in this workspace (the root
is not itself a module — the same `directory prefix . does not contain
modules listed in go.work` failure the design doc records for
`go build ./...`, proven on the unmodified tree). `go vet ./...` was
therefore run **per module**, from each of the 91 discovered `go.mod`
directories, in the workspace:

```
find services libs -name go.mod -not -path '*/node_modules/*' -print0 \
  | while IFS= read -r -d '' modfile; do
      dir="$(dirname "$modfile")"
      ( cd "$dir" && go vet ./... )
    done
```

Per-module exit codes: **91/91 modules exit 0.** Zero `go vet` findings
anywhere in the workspace, before or after the fmt sweep.

## 4. Step 3 — triage table

| file:line | linter | disposition | rationale |
|---|---|---|---|
| `services/atlas-channel/atlas.com/channel/character/processor_test.go:246` | staticcheck (QF1011) | excluded → `docs/TODO.md#lint-burn-down-task-171-follow-up` (new sub-bullet "task-261 staticcheck QF1011 exclusion") | The finding exists only because the gofumpt fix removed a redundant paren pair on this exact line, moving it under `--new-from-rev`. The line is a compile-time signature assertion: `var _ func(character.Model) character.Model = mock.NewMockProcessor().PartyDecorator`. QF1011's suggested rewrite, `var _ = mock.NewMockProcessor().PartyDecorator`, still compiles for *any* signature `PartyDecorator` has — it silently drops the assertion that the method matches `func(character.Model) character.Model`. That is a behavioral change to what the test verifies, not a mechanical formatting fix, so per the Step 3 policy table it takes the exclusion path: `.golangci.yml` carries a path-scoped `staticcheck` / `QF1011` exclusion with a comment naming this burn-down entry, and `docs/TODO.md` carries the entry describing the durable fix (rewrite the assertion in a form staticcheck accepts without weakening it, then remove the exclusion). |

No other finding was produced by either `tools/lint.sh --check --go` or the
per-module `go vet ./...` sweep. 90 of 91 lint-layer module runs, and all 91
vet-layer module runs, reported zero findings — see §5 below for the
verbatim zero-finding output.

## 5. Step 4 — re-run to green

After adding the `.golangci.yml` exclusion, `services/atlas-channel` was
re-checked in isolation first:

```
tools/lint.sh --check --go services/atlas-channel
```

```
0 issues.
lint.sh: OK
EXIT=0
```

Then the whole-tree check was re-run:

```
tools/lint.sh --check --go
```

Tail of the captured output (all 91 modules report `0 issues.`, all fmt
targets clean):

```
0 issues.
0 issues.
0 issues.
0 issues.
0 issues.
0 issues.
0 issues.
0 issues.
0 issues.
0 issues.
0 issues.
0 issues.
0 issues.
lint.sh: OK
LINT_EXIT=0
```

91 of 91 modules in the log read `0 issues.`; final line `lint.sh: OK`;
**`LINT_EXIT=0`**.

`go vet ./...` was re-run per module exactly as in §3 with no source changes
since the first pass; all 91 modules still exit 0.

## Success condition reached

`tools/lint.sh --check` (whole tree, both layers) exits 0 with zero failing
fmt and zero failing lint targets. This is the state Task 10 cites.

## Side effect observed and not committed

Running `tools/lint.sh` modified `go.work.sum` (net 1 insertion / 4 deletions
of stale, no-longer-referenced checksum lines — pre-existing pruning drift,
not something this task's source edits caused). Per the controller's
instruction this was left uncommitted; `git status` after this task's commit
still shows `go.work.sum` modified in the worktree.
