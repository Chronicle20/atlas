# Review — Task 1 & 2 (commits `b1b64f67c`, `64e6b9368`)

Brief: `docs/tasks/task-266-process-parity-agent-rename/prd.md` (§4 file
inventory, §10 acceptance criteria). Canonical spec: `docs/process-parity.md`.

Scope reviewed: the diff of `b1b64f67c^..64e6b9368` (18 files, 69
insertions/58 deletions), plus the current content of every file that diff
touches or whose contract the diff depends on (`docs/process-parity.md`,
`docs/agent-dispatch.md`, `.claude/settings.json`, `tools/agent-ledger.sh`,
`tools/task-step.sh`). Did not re-verify the three items the controller marked
already independently verified (content-neutral rename, §3.1 hooks clean,
frontmatter/filename match) but spot-confirmed each did not visibly regress
while reading adjacent hunks.

## Findings

### 1. Narrowed §7 check 3 does not actually pass for `atlas` — BLOCKING

`docs/process-parity.md:247-250` (added in `64e6b9368`) reads:

> Each repository defines `task-implementer`, `task-verifier`, and
> `task-reviewer`, and no reference to `atlas-implementer`, `atlas-verifier`,
> or `atlas-reviewer` remains on the live surface — `.claude/`, `CLAUDE.md`,
> top-level `docs/*.md`, and `tools/`. Historical records under `docs/tasks/`
> are exempt (see `docs/agent-dispatch.md` for the cutover).

Running exactly what this check specifies against the current tree:

```
$ git grep -lE 'atlas-(implementer|verifier|reviewer)' | grep -v '^docs/tasks/'
docs/agent-dispatch.md
docs/process-parity.md
```

Both hits are inside the check's own named surface (`docs/agent-dispatch.md`
and `docs/process-parity.md` are both top-level `docs/*.md`), and neither is
under the `docs/tasks/` exemption the check carves out. So the check as
literally written *fails* for `atlas` today — the one repository that has
already completed the rename this check exists to verify. The two remaining
hits are deliberate and legitimate (the historical-name-cutoff paragraph in
`docs/agent-dispatch.md:8-14`, which necessarily names the old strings to
explain them, and `docs/process-parity.md`'s own §2/§3.5/§5.1 rename-mapping
prose describing the decision for the three repos that haven't ported it
yet) — but the check text carves out neither. This is exactly the same class
of defect PRD §4.4 asked Task 2 to fix ("anywhere" was unsatisfiable because
of `docs/tasks/`); the amendment narrowed the domain but did not add the two
additional carve-outs it needed (the cutover note itself, and the canonical
spec document's own rename-history prose) to become actually satisfiable.
This is a real gap in the Task 2 authored edit, not a pre-existing PRD issue —
the check is new text written in this commit.

The implementer flagged this exact tension in
`docs/tasks/task-266-process-parity-agent-rename/reports/task-1-2.md`
("Note on the two remaining `atlas-` hits", "Issues / concerns") and asked
the controller to confirm the intended reading. That flag is honest and
useful, but the underlying check text is still wrong as committed — a future
mechanical run of check 3 (e.g. by a `home-hub`/`Harbormaster`/`MyFleet`
phase-4 closer, or a script) will report `atlas` as not-done when it is done.
Fix: add an explicit carve-out for `docs/process-parity.md` (the canonical
spec, not a repo's own procedures) and for the cutover paragraph in
`docs/agent-dispatch.md`, or reword the check to match strings only where
they don't appear inside a sentence that also contains "renamed"/"historical".

### 2. PRD §10 acceptance criterion is unsatisfied by the same gap — non-blocking (PRD, not implementation, issue)

PRD §10: "`git grep -lE 'atlas-(implementer|verifier|reviewer)'` returns only
paths under `docs/tasks/`." As shown above, it does not — it also returns
`docs/agent-dispatch.md` and `docs/process-parity.md`. This is the same root
cause as finding 1, and PRD §4.4 explicitly *required* the addition that
causes the AC to fail (the cutover note in `docs/agent-dispatch.md` naming the
old strings), so the AC line is internally inconsistent with §4.4 rather than
a fixable implementation error. Not blocking on its own — grouped here so the
controller doesn't re-file it as a new finding — but the PRD's checklist
should be corrected before it's used as a completion gate.

## Checks that passed

- **Dangling wiring.** `.claude/settings.json:8-56` matches hooks by tool
  type (`Write|Edit`, `Agent`, `Bash`, `*`), never by agent name — no runtime
  wiring depends on the old strings. `.claude/commands/execute-task.md` and
  `.claude/commands/fix-pr-bug.md` consistently use `task-implementer` /
  `task-verifier` / `task-reviewer` in every dispatch instruction and
  `--agent-type` flag (checked via grep, all 16 hits renamed, zero `atlas-`
  residue). `tools/agent-ledger.sh:36` and `tools/task-step.sh:10` only
  changed help text / a comment; grepped both files for any logic branching
  on the literal string — none exists, confirming PRD §4.2's claim.
  `.claude/hooks/commit-boundary.sh`, `turn-budget-guard.sh`, `turn-budget.sh`
  changed only prose/comments (lines 106, 126, 98, 26 respectively) — no
  conditional logic touches the renamed strings.
- **Historical boundary.** `git diff --stat b1b64f67c^..64e6b9368 -- docs/tasks/`
  is empty — nothing under `docs/tasks/` was touched by either commit.
- **Two of the three authored edits are correct and self-consistent.** The
  `CLAUDE.md` owner-table row (`CLAUDE.md:85` after the edit) points at
  `docs/process-parity.md`, which exists. The historical-name-cutoff note in
  `docs/agent-dispatch.md:8-14` accurately describes the rename and correctly
  attributes old-name rows in `docs/tasks/` to pre-task-266 dispatches. (The
  third authored edit, the §7 check-3 narrowing, is finding 1 above.)
- **Line endings.** `grep -c $'\r'` returns 0 for all 18 files touched across
  both commits — no CRLF introduced or normalized.
- **Tests.** `tools/agent-ledger_test.sh`, `.claude/hooks/wait-loop-guard_test.sh`,
  and `tools/task-numbers_test.sh` all pass (exit 0) on this branch, matching
  PRD §10's third-from-last acceptance line.

## Not evaluable

- Whether `docs/process-parity.md` §7 check 3's eventual mechanical
  evaluation for `home-hub`, `Harbormaster`, or `MyFleet` will behave
  correctly once those repos port `agent-dispatch.md` — those repos don't
  exist in this worktree and porting is out of scope for phase 1 (PRD §2
  non-goals). Flagged as a design concern in finding 1, not verified against
  a real second repository.

## Verdict rationale

One real, in-scope defect (finding 1): a check written in this unit's own
second commit does not hold for the state that same commit produces. It does
not break any runtime behavior — `atlas`'s dispatch surface is fully renamed
and functions correctly — but it is a wrong assertion landing in the
canonical cross-repo spec, which is exactly the kind of thing §7 says must be
"mechanically checkable" and isn't, as written. That is a blocking finding,
not a nitpick: leaving it in means a future phase or automated pass will
mis-evaluate `atlas`'s own status.
