# Review: commit a6a25b079 "chore(process): apply the 2026-08-29 weekly session-audit recommendations"

Branch: `chore/audit-week-0829-config`
Reviewed against: commit message + `~/.claude/audits/week-2026-08-29-consolidated.md` "Recommendations, ranked" (report not committed — confirmed absent from the diff).

## Scope confirmed

`git show --stat a6a25b079` touches exactly the 14 files the commit message and
task brief describe: `packet-verifier.md` + `VERIFYING_A_PACKET.md`,
`execute-task.md` + `plan-task.md` (model pin), `commit-boundary.sh` (comment
only), `context-handoff-guard.sh` (+ test, new), `wait-loop-guard.sh` (+ test),
`format-on-write.sh`, `spec-task.md`, `settings.json`, `docs/agent-dispatch.md`,
`docs/process-parity.md`. No unrelated files. No literal home/absolute paths in
the diff (`grep -n '/home/\|/Users/'` on the full commit: no matches). Working
tree is clean at HEAD. Scope matches the stated work.

## Findings

### Blocking

1. **`docs/process-parity.md:70` is stale for `format-on-write.sh`.** The
   table still lists the pre-commit line count (`45`) and the pre-commit
   description ("Hardcodes `services/atlas-ui` for prettier and sources
   `tools/toolchain.versions`..."). The file this commit ships is 67 lines
   (`wc -l .claude/hooks/format-on-write.sh` → 67) and gained an entire new
   PostToolUse-Bash code path (in-place `.go` edit detection, 20-file cap).
   The table entry was not touched by this commit (confirmed via `git show
   a6a25b079~1:docs/process-parity.md` — identical `45` and same prose), so
   this is an omission in this commit, not a pre-existing gap. This is
   exactly the doc/rule-consistency check the review priorities called out,
   and it fails: the portable-hooks line-count table is now wrong for a hook
   this same commit changed. (By contrast, `wait-loop-guard.sh` (157) and the
   new `context-handoff-guard.sh` (82) entries in the same table are correct
   — `wc -l` confirms both.)

2. **`format-on-write.sh`'s new Bash path has no repo-containment check —
   it can format a file outside the repo.** `format_one()` (lines 19-44)
   only requires an absolute path (`case "$fp" in /*) ;; *) return 0 ;; esac`,
   line 25); it never checks the resolved path is under `$ROOT`. The new Bash
   trigger (lines 56-65) extracts every bare `.go`-suffixed token from the
   command text and, for relative tokens, resolves them against
   `tool_input.cwd` (line 62: `p="${cwd:-$ROOT}/$tok"`), then calls
   `format_one "$p"`. `cwd` is whatever directory the Bash tool call actually
   ran in — nothing here confirms it is inside the atlas repo. A command such
   as `cd /some/other/checkout && sed -i 's/x/y/' pkg/file.go` (cwd outside
   `$ROOT`) or one that names an absolute path outside the repo directly
   (e.g. `sed -i '' /tmp/scratch/file.go`) matches the trigger regex and will
   have `golangci-lint fmt` run against it, in-place, from that file's own
   directory upward looking for a `go.mod`. This is the specific property the
   review priorities asked to verify ("cannot format a file outside the
   repo") and it does not hold for the new path. It is fail-open in the sense
   that it never blocks the session (still ends in `exit 0`), but it silently
   rewrites a file the hook was never asked to touch — a materially larger
   blast radius than the pre-existing Write/Edit path (which only ever acts
   on a `file_path` the harness itself is actively editing, not one
   text-matched out of an arbitrary shell command with a caller-supplied
   `cwd`). Needs a `case "$fp" in "$ROOT"/*) ;; *) return 0 ;; esac`-style
   containment check before either format branch runs.

### Non-blocking

3. **`format-on-write.sh:29,31` — `exit 0` inside `format_one()`, not
   `return 0`.** The function was refactored from straight-line script to a
   function (this commit), and the two `.go`-branch early-outs
   (`source ... || exit 0` and `[ -x "$GOLANGCI" ] || exit 0`) were left as
   `exit 0` while the sibling checks at lines 21, 25, 37 were correctly
   changed to `return 0`. `exit 0` inside a bash function still terminates
   the whole process, not just the function, so in the new
   call-`format_one`-in-a-loop path (lines 61-65, up to 20 `.go` tokens) a
   missing `toolchain.versions` or missing cached `golangci-lint` binary
   aborts the entire hook on the first file and never even reaches the
   remaining tokens. In the current code this happens to be harmless because
   both conditions are file-independent within one invocation (same `$ROOT`,
   same `GOLANGCI_LINT_VERSION` for every token, so if it fails for file 1 it
   would fail identically for files 2-20 via `return 0` too) — but it is a
   latent trap for the next person who adds a per-file check in that branch,
   and it is inconsistent with the rest of the function. Worth a follow-up
   fix, not blocking this commit.

4. **`wait-loop-guard.sh`'s pipe-stage splitter can be defeated by a quoted
   `|`.** Lines 78-83 split the normalized command on literal `|` via
   `IFS='|'`, so `grep -E 'a|b' file` is split into stages `grep -E 'a`,
   `b' file`, neither of which matches `readonly_re` — the command is
   classified as non-read-only and the poll streak resets. This only
   *under*-counts polling (a legitimate command using an internal `|` inside
   quotes will never trip the 3rd-repeat deny, and will also never let a
   truly repeated poll accumulate past it if that poll happens to contain a
   quoted pipe) — it cannot produce a false-positive deny of a legitimate
   command, only a missed detection. Not blocking given the review priority
   was false-positive wedges, but worth noting since the docstring's own
   example (`grep -c 'EXIT=' log`) has no quoted pipe and so is unaffected in
   practice.

## Verified PASS (with evidence)

- **ESCALATE resolution.** `commit-boundary.sh:38` is `ESCALATE=60`;
  `context-handoff-guard.sh`'s `sed -n 's/^ESCALATE=\([0-9][0-9]*\).*/\1/p'`
  against that file returns `60` (verified by running the same sed against
  the literal line). The `case "${ESCALATE:-}" in ''|*[!0-9]*) ESCALATE=60 ;;
  esac` fallback is sane and matches the real value, so a parse failure would
  silently coincide with the current correct value.
- **State-file key collision with `turn-budget.sh`/`turn-budget-guard.sh`.**
  Confirmed no collision: `turn-budget.sh:55` writes counters at
  `$state_dir/agent-$agent` or `$state_dir/session-$session` (no prefix
  beyond the role tag); `context-handoff-guard.sh` reads the exact same path
  (`${TMPDIR:-/tmp}/claude-turn-budget/session-$session`). `wait-loop-guard.sh`'s
  new poll-streak file is `$state_dir/poll-$key` where `$key` is
  `agent-<id>`/`session-<id>` — i.e. `poll-agent-<id>` / `poll-session-<id>`,
  a distinct filename from the counter files, verified by reading
  `turn-budget.sh:54-55`, `turn-budget-guard.sh:53-54`, and
  `wait-loop-guard.sh:67,71`. `turn-budget.sh`'s daily prune
  (`find "$state_dir" -type f -mtime +1 -delete`) is a blanket sweep by
  mtime, not a name-based operation, so it prunes stale `poll-*` files the
  same way it prunes stale counters — symmetric, not a clobber.
- **`context-handoff-guard.sh` never touches subagents.** `agent="$(...
  jq -r '.agent_id // ""' ...)"`; `[ -z "$agent" ] || exit 0` (line 20)
  exits before any counter read whenever `agent_id` is present. Confirmed by
  the test suite's "subagents and unkeyed calls are never blocked" case
  (agent-abc at count 200 still allowed).
- **`context-handoff-guard.sh` allows sessions with no counter file.**
  `[ -f "$counter" ] || exit 0` (line ~31); test case "no counter file:
  silent" passes.
- **Finishing-dispatch allowlist matches the repo's actual reviewer/auditor
  agent set.** `ls .claude/agents/` = backend-guidelines-reviewer,
  frontend-guidelines-reviewer, family-auditor, packet-completeness-critic,
  plan-adherence-reviewer, service-documentation, task-reviewer,
  task-verifier, todo-scanner, plus the implementer/verifier agents
  (task-implementer, packet-implementer, dispatcher-family-implementer,
  packet-verifier) which are correctly *absent* from the allowlist (they are
  new-unit dispatches and should be denied past threshold — confirmed
  covered by the test suite's "past threshold: new units denied" block using
  `task-implementer`, `general-purpose`, `Explore`, `packet-implementer`,
  `Plan`).
- **`CONTEXT-JUSTIFIED:` escape hatch.** `grep -q 'CONTEXT-JUSTIFIED:'`
  against `prompt + description`; test case "past threshold: justified
  exception allowed" passes.
- **Both hooks are silent and exit 0 on the happy path.** Read end-to-end;
  every branch either falls through to `exit 0` or emits a `deny` JSON
  object then `exit 0`; no branch returns non-zero, and `jq -nc ... || exit 0`
  guards a malformed-JSON emission from producing garbage stdout.
- **Both test suites pass.** `bash .claude/hooks/wait-loop-guard_test.sh` →
  `passed: 63 failed: 0`, exit 0. `bash .claude/hooks/context-handoff-guard_test.sh`
  → `passed: 16 failed: 0`, exit 0.
- **Test suites do not pass "for the wrong reason."** Each `allow`/`deny`
  helper asserts on the actual `permissionDecision` JSON field (or its
  absence), not merely on the hook's exit code, so a hook that errored out
  before deciding would produce empty stdout and be indistinguishable from a
  real "allow" only in the allow-cases — but the deny-cases explicitly grep
  for `"permissionDecision":"deny"`, which a crashed/errored hook would never
  emit. Spot-checked by temporarily reasoning through the "no counter file"
  and "subagent" allow-cases: both reach an explicit early `exit 0` in the
  hook (not a crash), consistent with the reason given in the test names.
- **`wait-loop-guard.sh`'s streak state does not leak across the new test
  block's cases in a way that makes an assertion vacuous.** The new block
  (`wait-loop-guard_test.sh:84-113`) uses a fresh `mktemp -d` exported as
  `$TMPDIR` for the whole file (line 85), and differentiates state purely by
  the `session_id`/`agent_id` keys it passes per case (`s1`, `s2`/agentX,
  `s3`, `s4`, and the no-id case) rather than by directory — each key gets
  its own `poll-<key>` file, so `s1`'s 3rd-deny does not affect `s2`'s
  independent 2-allow/1-deny sequence, and the final stateless case (no
  session/agent id, line 113) is a distinct code path (`key=""` short-circuits
  before any file I/O) rather than reusing `s1-s4`'s state.
- **Settings.json is valid JSON** (`jq empty` exits 0) and the hook wiring is
  as described: `context-handoff-guard.sh` added to the existing `Agent`
  `PreToolUse` matcher alongside `fork-dispatch-guard.sh`; `wait-loop-guard.sh`
  unchanged on its own `Bash` `PreToolUse` matcher; the `PostToolUse` matcher
  for `format-on-write.sh` widened from `Write|Edit` to `Write|Edit|Bash`.
  `commit-boundary.sh` remains on its own separate `PostToolUse`/`Bash` hook
  entry — two independent hook registrations firing on the same event/tool
  is normal (each hook decides independently), not a double-fire of the same
  script, and `format-on-write.sh`'s own internal branching (Write/Edit
  `file_path` vs. Bash `command`) prevents it from doing anything on a Bash
  call that also happens to be a Write/Edit (mutually exclusive input shapes).
- **20-file cap in `format-on-write.sh`'s Bash path is present and correct.**
  `n=$((n + 1)); [ "$n" -ge 20 ] && break` (line 64) inside the `.go`-token
  `for` loop; bounded, no risk of runaway iteration since the token list
  itself is already a finite `grep -oE` extraction from one command string.
- **`model: sonnet` frontmatter is syntactically valid YAML.** Confirmed by
  reading the raw file: the comment block and `model: sonnet` line sit
  between the pre-existing `---` frontmatter delimiters in both
  `execute-task.md` and `plan-task.md`, not after the closing `---` — `#`
  comments are valid inside a YAML block, so this is a well-formed addition,
  not stray prose leaking into the command body.
- **`packet-verifier.md` and `VERIFYING_A_PACKET.md` changes match the
  brief exactly**: 6-cell batch cap stated in both files consistently, "run
  matrix/--check once per batch" stated in both, and the continuation-brief
  language ("do NOT re-read the playbook whole... re-read only the sections
  the brief names as still open") is present in `packet-verifier.md`.
- **`spec-task.md` Phase 1 dispatches no research agents** — the added
  paragraph explicitly forbids fanning out into WZ/IDA/registry/pipeline
  research during Phase 1 and redirects unanswered questions to §9 Open
  Questions.
- **`plan-task.md` "edit in place"** — the added paragraph explicitly
  forbids `plan-b.md`/`plan-c.md` siblings and instructs editing `plan.md`
  directly; no stray reference to plan-b/plan-c siblings remains elsewhere
  in `plan-task.md` or `docs/`.
- **No literal home/absolute paths land in any committed file** (checked the
  full commit diff for `/home/` and `/Users/`; no matches).
- **Report source not committed.** `~/.claude/audits/week-2026-08-29-consolidated.md`
  does not appear anywhere in the diff or `git show --stat`.
- **Commit message accuracy.** All seven numbered changes in the commit body
  are backed by a corresponding diff hunk; nothing claimed is unsupported.
  (The `commit-boundary.sh` 3-line comment-only change and the two doc
  updates in `docs/agent-dispatch.md`/`docs/process-parity.md` are not
  separately enumerated in the commit message, but they are directly
  supporting the `context-handoff-guard.sh` and `wait-loop-guard.sh` items
  that are enumerated — not scope creep.)

## Not evaluable

- Live behavior of `context-handoff-guard.sh` and `wait-loop-guard.sh` inside
  an actual Claude Code session (only the shell-level unit tests and manual
  JSON-payload runs were exercised here, per the review's own scope — this
  matches how the task brief scoped the check).
- Whether `execute-task.md`/`plan-task.md`'s `model: sonnet` pin actually
  changes controller behavior in a live dispatch — frontmatter parsing by
  the harness is outside this commit's diff and outside repo source under
  review.

## Ranked recommendation

Both blocking findings are in `format-on-write.sh` and its accompanying doc
table — the two shell guards (`context-handoff-guard.sh`, `wait-loop-guard.sh`)
and their test suites are solid: no false-positive denial path was found, no
state-file collision, ESCALATE resolves correctly, and both hooks are silent
and exit 0 on every path exercised. The format-hook containment gap (#2) is
the more consequential of the two — it lets a Bash-tool call whose `.go`
tokens or `cwd` land outside the repo trigger an in-place reformat outside the
repo tree, which the review was explicitly asked to verify does not happen.
