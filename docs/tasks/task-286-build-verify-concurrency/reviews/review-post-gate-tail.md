# Review: post-gate tail (6080dc121..HEAD), task-286

Scope: the tail of task-286 after the final implementation gate (6080dc121),
covering audit-trail docs, measurement records, a restart-handoff note added
then deleted, and one functional change (gitleaks allowlist fix).

Commits reviewed:
- `dde9f32df` docs(task-286): commit the review and audit trail
- `96462a71d` fix(gitleaks): allow angle-bracket Windows profile placeholders
- `d8ff0381b` docs(task-286): record the branch-end criterion-2 before/after fan-out measurement
- `dcee44651` docs(task-286): restart handoff for the post-tuning session
- `b92f8c341` docs(task-286): record Layer 0 after-figures post host tuning
- `d149b8b5f` docs(task-286): remove pre-restart handoff note

`git diff --stat 6080dc121..HEAD`: 19 files changed, 3160 insertions(+), 12
deletions(-) — `.gitleaks.toml` (+4) plus 18 doc files under
`docs/tasks/task-286-build-verify-concurrency/` (audit trail, per-task
reviews, `measurements.md`, `agent-ledger.tsv`). No source/test code besides
the gitleaks config.

## 1. `96462a71d` — gitleaks allowlist scoping

`.gitleaks.toml` added one allowlist regex:

```
'''[A-Za-z]:\\Users\\<[^>\\]+>''',
```

The pre-existing `windows-user-path` rule (`[A-Za-z]:\\Users\\[^\\\s"']+`)
matches up to the next backslash/whitespace/quote, so for the mandated doc
placeholder `C:\Users\<windows-user>\.wslconfig` its match text is exactly
`C:\Users\<windows-user>`. The new allowlist regex matches that same
substring (requires `<...>` immediately after `\Users\`, no backslash inside
the brackets) and nothing broader.

Verified empirically against the actual installed `gitleaks` binary
(v8, default `regexTarget` = matched secret, not full line — confirmed via
`.github/workflows/secret-scan.yml:54-62`, no `regexTarget` override in
config):

The fixture held two lines: the bracketed doc placeholder
(`C:\Users\<windows-user>\.wslconfig`) and the same path with the brackets
removed so the profile segment is a bare literal name.

```
$ gitleaks detect --no-git --source /tmp/gltest --config .gitleaks.toml --redact -v
Doc placeholder line: -> no finding
Bare literal line:    -> 1 finding (windows-user-path)
```

The placeholder is silenced; a literal profile name is still caught.
The change is narrowly scoped to the bracketed-placeholder shape and does not
open a hole for real secrets. **PASS.**

The three consumers of the placeholder text that motivated this fix
(`design.md:104`, `plan.md:247`, `review-task-2.md:63`) predate this range
and were not touched by it — consistent with "fix the false positive the
existing docs already tripped," not a scope expansion.

## 2. `measurements.md` — Layer 0 "### After" section (added in `b92f8c341`)

Diff: `docs/tasks/task-286-build-verify-concurrency/measurements.md:134-182`
(new "### After" subsection) plus a new `## Criterion 2 …` section appended
at end of file (`d8ff0381b`, lines ~600-671).

Checked for internal contradiction across the file:

- **Before/after figures are consistent with each other.** Before: `tmpfs
  16G … 33% /tmp`, `Mem: 31Gi total`, no `/etc/fstab` `/tmp` line. After:
  `tmpfs 4.0G … 0% /tmp`, `Mem: 50Gi total`, `nproc` 24, `/etc/fstab` carries
  the pinned `size=4G` line. The narrative directly ties the before figure
  ("31 GiB") to the `## Host tuning (WSL2)` section's stated default and the
  after figure to the applied `.wslconfig` (`memory=52GB`) — no numeric
  mismatch found.
- **Ordering across the two new sections is internally coherent.** The
  Criterion-2 section (branch-end fan-out timing) explicitly states it was
  run **before** the WSL restart ("Host state at measurement time:
  untuned … the WSL restart that activates it was deliberately deferred
  until after this measurement"), and the Layer 0 "After" section is dated
  the same day but explicitly *after* `wsl --shutdown`. The two sections
  agree on sequencing rather than contradicting each other on which host
  state was current when.
- **Deviations from the brief are disclosed, not hidden.** Two are called
  out by name in the "After" subsection: (a) `tools/scratch-sweep.sh --now
  --root /tmp` exits 2 because `/tmp` is on the script's dangerous-root
  refusal list, so the sweep was run against the default root instead
  (`/var/tmp/atlas/scratch`); (b) the `/tmp` after-figures therefore reflect
  the VM restart's fresh tmpfs, not an actual sweep of `/tmp`. A third
  observed anomaly is also disclosed rather than silently corrected: the
  operator's `/etc/fstab` pin line is present twice (duplicate), noted as
  "harmless … but one copy should be removed" rather than edited out of the
  record. This matches the review-protocol expectation that deviations are
  surfaced, not papered over. **PASS.**
- One operational hazard is documented rather than omitted: `--now` deleted
  the recording session's own live scratch mid-run; recorded as a note that
  `--now` should not be run with active sessions running. Good-faith
  disclosure, not a defect in this doc.

No contradiction found between the newly added sections and the pre-existing
content of `measurements.md` (Layer 2's earlier `nproc = 24` reference is
consistent with the Layer 0 before/after figures — processor count did not
change with this tuning pass; only memory/swap and `/tmp` sizing did).

## 3. Literal home/absolute paths in added lines

Swept the full range's added lines:

```
git diff 6080dc121..HEAD -- '*.md' | grep '^+' | grep -iE '/home/[a-z]|Users\\\\[a-zA-Z]+\\\\|C:\\\\Users\\\\[a-zA-Z]' | grep -v '<windows-user>|/home/runner|/home/builder'
→ no matches
git diff 6080dc121..HEAD | grep '^+' | grep -i 'tumidanski|/home/'
→ one match, line commenting on a *verification command* ("No literal home
  path: git diff … | grep -E '/home/'"), not a literal path itself.
```

No literal home/absolute-path violations found in the range's added content.
**PASS.**

## 4. `restart-handoff.md` fully gone from HEAD

```
git show dcee44651 --stat   → adds restart-handoff.md (+76)
git show d149b8b5f --stat   → deletes restart-handoff.md (-76), same line count
ls docs/tasks/task-286-build-verify-concurrency/ | grep -i restart → no output
find <worktree> -iname "restart-handoff*" → no output
```

Confirmed fully removed at HEAD; the add/delete pair nets to zero content in
the working tree. **PASS.**

## Non-blocking notes

- `dcee44651` and `d149b8b5f` (the restart-handoff add/remove pair) carry
  `Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>`. `CLAUDE.md`
  states "Never use Fable for background or review workflows." These are
  scratch session-continuity notes rather than review/background-agent work
  product, and the note was deleted before landing in this range, so I am
  not treating it as blocking — flagging for awareness only.
- The working tree at review time carries unrelated uncommitted noise
  (`services/zz-verify-probe-broken-23184/`, `tools/zz-verify-probe-*`,
  a modified `go.work.sum`, a modified `agent-ledger.tsv`) that is not part
  of the `6080dc121..HEAD` commit range. Out of scope for this review per
  the reviewer protocol (do not widen scope beyond the given range); noting
  only so it isn't mistaken for part of the reviewed diff.

## Not evaluable

- None. All four review items were directly checkable within the given
  range using tool output (gitleaks binary run against a synthetic
  reproduction, file diffs, `ls`/`find`).

## Verdict rationale

All four focus items pass with cited evidence. No blocking defects found in
the tail range. Two non-blocking observations recorded above.
