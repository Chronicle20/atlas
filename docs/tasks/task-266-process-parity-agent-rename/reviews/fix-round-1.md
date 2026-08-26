# Review — Fix round 1 (commit `e75c2a168`)

Brief: prior blocking finding recorded at
`docs/tasks/task-266-process-parity-agent-rename/reviews/task-1-2.md`, finding
1 — `docs/process-parity.md` §7 check 3 described a check that did not
actually pass on this branch, because two files legitimately retain
`atlas-*` agent-name references (`docs/agent-dispatch.md`'s historical-cutoff
note, and `docs/process-parity.md`'s own rename-mapping prose) and both sit
inside the "live surface" the check named.

Scope reviewed: the diff of commit `e75c2a168` only (5 files: `docs/process-
parity.md`, this task's `plan.md`, `prd.md`, `reports/task-1-2.md`,
`reviews/task-1-2.md`). Earlier commits `b1b64f67c`/`64e6b9368` are out of
scope per the review brief and were not re-reviewed.

## 1. Does §7 check 3 now specify a command that actually passes on this branch?

Yes, verified by running it, not by trusting the commit message.

`docs/process-parity.md:252-256` (current text):

```sh
git grep -lE 'atlas-(implementer|verifier|reviewer)' -- . ':!docs/tasks' \
  | grep -vxE 'docs/(agent-dispatch|process-parity)\.md'
```

Ran verbatim from the repo root of this worktree:

```
$ git grep -lE 'atlas-(implementer|verifier|reviewer)' -- . ':!docs/tasks' \
    | grep -vxE 'docs/(agent-dispatch|process-parity)\.md'
(no output, grep exit 1 — "no lines selected", which is the correct outcome
 for "must print nothing")
```

The check now passes as literally written, for this repo, today. PASS.

## 2. Are the two carve-outs justified, or too broad?

Justified and narrowly scoped. `grep -vxE 'docs/(agent-dispatch|process-
parity)\.md'` uses `-x` (whole-line match against the path), so it excludes
exactly `docs/agent-dispatch.md` and `docs/process-parity.md` and nothing
else — it would not, for example, accidentally swallow a third file whose
path merely contains "agent-dispatch" as a substring.

Confirmed both excluded files contain only explanatory prose about the
rename, not a dangling operative reference:

```
docs/agent-dispatch.md:8-9   — "Historical-name cutoff. Task-266 renamed
                                 atlas-implementer, atlas-verifier, and
                                 atlas-reviewer to the generic ..."
docs/process-parity.md:38-39 — describes files that "currently hardcode
                                 atlas-implementer / atlas-verifier" (§3.1
                                 discussion of the pre-rename state)
docs/process-parity.md:77-79 — rename-mapping table, "task-implementer (was
                                 atlas-implementer)" etc.
docs/process-parity.md:185-186 — §5.1 rename-instruction prose naming the
                                 old→new mapping
```

All six hits are prose that names the old strings in order to explain the
rename; none is a place a tool, hook, or agent dispatch would actually invoke
the old name. No genuine dangling reference is hiding behind the carve-out —
confirmed by the exhaustive `git grep` above returning nothing once these two
files are excluded, i.e. there is no third file that would need scrutiny.
PASS.

## 3. Is the home-hub/Harbormaster/MyFleet reasoning checkable from this worktree?

No — flagged as **not evaluable**. The new sentence at
`docs/process-parity.md:257-260` asserts:

> In `home-hub`, `Harbormaster`, and `MyFleet` the `docs/agent-dispatch.md`
> exemption does not apply — those repositories never used the `atlas-*`
> names, so they need no historical-cutoff note and only
> `docs/process-parity.md` is exempt there.

This is a factual claim about the git history and current content of three
repositories that do not exist in this worktree or this repo's filesystem.
It cannot be run or grepped from here. On its face the reasoning is
internally consistent (if a repo's agent files were never named `atlas-*`,
there is nothing for a "historical cutoff" note to explain), but "on its
face consistent" is not the same as "true of those repos" — that is an
unverified factual premise the fix commit introduces without evidence
reachable from this scope. Recorded under Not evaluable, not treated as a
defect, per the review brief's explicit instruction not to assert something
uncheckable here.

## 4. Was `prd.md` §10 updated consistently with the spec?

Yes. `docs/tasks/task-266-process-parity-agent-rename/prd.md:183-190` now
reads:

> - [ ] The `docs/process-parity.md` §7 check 3 command prints nothing —
>   that is, the only surviving `atlas-*` agent references are historical
>   records under `docs/tasks/` plus the two documents that exist to explain
>   the rename, `docs/agent-dispatch.md` and `docs/process-parity.md`.

This matches the spec's carve-out exactly for this repo (atlas) — both
named exempt documents, same two-file scope, same "prints nothing" success
condition. The PRD does not restate the home-hub/Harbormaster/MyFleet
asymmetry from spec §7 check 3, but that asymmetry only matters to those
other repos' own phase-4 PRDs; task-266's PRD is scoped to this repo, where
the AC and the spec agree. No PRD/spec disagreement found. PASS.

## 5. Did the fix commit touch anything out of scope?

`git show e75c2a168 --stat --name-status`:

```
M  docs/process-parity.md
A  docs/tasks/task-266-process-parity-agent-rename/plan.md
M  docs/tasks/task-266-process-parity-agent-rename/prd.md
A  docs/tasks/task-266-process-parity-agent-rename/reports/task-1-2.md
A  docs/tasks/task-266-process-parity-agent-rename/reviews/task-1-2.md
```

No `.claude/` or `tools/` file touched. No `docs/tasks/` path outside this
task's own folder touched — confirmed separately for the whole branch:
`git diff --stat 02fd0d844..e75c2a168 -- docs/tasks` shows only this task's
four files. `docs/process-parity.md` is the one top-level doc touched, which
is exactly what the prior review's finding required. PASS — no scope
violation.

(Note, non-blocking: `reports/task-1-2.md` and `reviews/task-1-2.md` are
Task 1/2 artifacts from a prior session, first added to git in this fix
commit rather than in `64e6b9368` where the work they describe landed. They
are within this task's own folder so this is not a scope violation, but a
controller reviewing commit boundaries should be aware the report/review
pair postdates the commits they describe by one commit.)

## Verdict rationale

The one blocking finding from the prior round is closed: the check is now an
executable command, it passes when run against this branch today, and the
two carve-outs are both narrowly scoped and individually justified by
inspection of their content. The commit stays within its own task folder and
does not touch `.claude/` or `tools/`. The one residual item — the
home-hub/Harbormaster/MyFleet claim — is a factual assertion about repos
outside this worktree's reach; it is plausible on its face but not something
this review can confirm, so it is reported as not evaluable rather than
approved or rejected.

## Not evaluable

- `docs/process-parity.md:257-260` — the claim that `home-hub`,
  `Harbormaster`, and `MyFleet` never used the `atlas-*` agent names. Not
  checkable from this worktree; those repositories are not present here.
